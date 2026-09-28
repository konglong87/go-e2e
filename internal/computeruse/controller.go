package computeruse

import (
	"context"
	"errors"
	"sync"
	"time"
)

// One total budget for control queuing, acknowledgement, and action draining.
// Backend control methods must honor context (native also enforces its own timeout).
const controllerControlGrace = time.Second

// ImageReader transfers ephemeral PNG bytes without putting them in the domain
// observation, event, or transcript. Authorization is enforced by Controller.
type ImageReader interface {
	ObservationImage(context.Context, string) ([]byte, string, error)
}

// Controller is one approved host session's execution authority. Agent and UI
// adapters must share this instance; constructing a Tool never grants approval.
type Controller struct {
	session      *ComputerSession
	backend      Backend
	serial       chan struct{}
	control      chan struct{} // Pause/Resume only; Stop must never queue behind them.
	stopping     chan struct{} // Repeat Stop must not abort an earlier Stop acknowledgement.
	mu           sync.Mutex
	cancel       context.CancelFunc
	pausing      int
	pauseVersion uint64
}

func NewController(s *ComputerSession, b Backend) (*Controller, error) {
	if s == nil || b == nil {
		return nil, errors.New("computer session and backend are required")
	}
	return &Controller{session: s, backend: b, serial: make(chan struct{}, 1), control: make(chan struct{}, 1), stopping: make(chan struct{}, 1)}, nil
}
func (c *Controller) Session() *ComputerSession { return c.session }
func (c *Controller) authorize(owner SessionOwner, id string) error {
	if c.session.ID() != id || !c.session.Owns(owner) {
		return errors.New("computer session ownership mismatch")
	}
	return nil
}

// lockControllerGate waits for the whole operation, including receipt processing.
// Unlike acquire, a control-plane barrier does not install an action cancel func.
func lockControllerGate(ctx context.Context, gate chan struct{}) (func(), error) {
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-gate
		return nil, err
	}
	return func() { <-gate }, nil
}
func (c *Controller) acquire(ctx context.Context) (context.Context, func(), error) {
	release, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return nil, nil, err
	}
	c.mu.Lock()
	op, cancel := context.WithTimeout(ctx, time.Duration(MaxActionDurationMS)*time.Millisecond)
	c.cancel = cancel
	c.mu.Unlock()
	return op, func() { cancel(); c.mu.Lock(); c.cancel = nil; c.mu.Unlock(); release() }, nil
}
func (c *Controller) cancelAction() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
}
func (c *Controller) Capabilities(ctx context.Context, owner SessionOwner, id string) (Capabilities, error) {
	if err := c.authorize(owner, id); err != nil {
		return Capabilities{}, err
	}
	release, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return Capabilities{}, err
	}
	defer release()
	caps, err := c.backend.Capabilities(ctx)
	if err != nil {
		return Capabilities{}, err
	}
	if err := c.session.UpdateCapabilities(caps); err != nil {
		return Capabilities{}, err
	}
	return caps, nil
}
func (c *Controller) Observe(ctx context.Context, owner SessionOwner, r ObserveRequest) (Observation, error) {
	if err := c.authorize(owner, r.SessionID); err != nil {
		return Observation{}, err
	}
	op, release, err := c.acquire(ctx)
	if err != nil {
		return Observation{}, err
	}
	defer release()
	if err = c.session.CanObserve(); err != nil {
		return Observation{}, err
	}
	o, err := c.backend.Observe(op, r)
	if err != nil {
		_ = c.session.Pause()
		return Observation{}, observeFailure(err)
	}
	if err = c.session.SetObservation(o); err != nil {
		_ = c.session.Pause()
		return Observation{}, err
	}
	return o, nil
}
func observeFailure(err error) error {
	var coded interface{ Code() string }
	if errors.As(err, &coded) {
		switch coded.Code() {
		case "unsupported_display":
			return errors.New("computer target is stale; refresh capabilities and observe again")
		case "permission_required":
			return errors.New("computer permissions are required; restore Screen Recording and Accessibility, then refresh capabilities")
		case "focus_changed":
			return errors.New("computer focus changed; observe again before continuing")
		}
	}
	return errors.New("computer capture failed; session paused")
}

func (c *Controller) Execute(ctx context.Context, owner SessionOwner, a Action) (ActionReceipt, error) {
	if err := c.authorize(owner, a.SessionID); err != nil {
		return ActionReceipt{}, err
	}
	op, release, err := c.acquire(ctx)
	if err != nil {
		return ActionReceipt{}, err
	}
	defer release()
	before, _ := c.session.CurrentObservation()
	if err = c.session.BeginAction(a); err != nil {
		return ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: OutcomeRejected, Verification: VerificationNotChecked, RedactedActionSummary: a.RedactedSummary(), ErrorCode: "action_rejected", CompletedAt: time.Now()}, err
	}
	started := time.Now()
	receipt, backendErr := c.backend.Execute(op, a)
	// No response or an invalid response after dispatch is never evidence of no
	// side effect. Always retain an unknown receipt, without backend error text.
	if receipt.ActionID != a.ID || receipt.SessionID != a.SessionID || !receipt.IsTerminal() {
		receipt = ActionReceipt{Outcome: OutcomeUnknown, Verification: VerificationUnknown, ErrorCode: "backend_outcome_unknown"}
	}
	receipt.ActionID = a.ID
	receipt.SessionID = a.SessionID
	caps := c.session.Capabilities()
	receipt.Platform = caps.Platform
	receipt.Backend = caps.Backend
	receipt.BeforeObservationID = before.ID
	receipt.Before = &before.Screenshot
	receipt.RedactedActionSummary = a.RedactedSummary()
	receipt.ErrorMessage = ""
	receipt.CompletedAt = time.Now()
	receipt.Duration = time.Since(started)
	if err = c.session.RecordReceipt(receipt); err != nil {
		return receipt, err
	}
	if backendErr != nil {
		return receipt, errors.New("computer backend failed; inspect receipt before continuing")
	}
	if receipt.AfterObservationID == "" || receipt.After == nil {
		// Losing visual evidence stops all following input, even if posting itself
		// was acknowledged as executed. Do not turn dispatch into verification.
		_ = c.session.Pause()
	}
	return receipt, nil
}
func (c *Controller) ObservationImage(ctx context.Context, owner SessionOwner, id, observationID string) ([]byte, string, error) {
	if err := c.authorize(owner, id); err != nil {
		return nil, "", err
	}
	if !c.session.Approved() {
		return nil, "", errors.New("computer observation is not authorized")
	}
	reader, ok := c.backend.(ImageReader)
	if !ok {
		return nil, "", errors.New("computer image reader is unavailable")
	}
	return reader.ObservationImage(ctx, observationID)
}

// beginPause revokes authority before waiting for any control or action gate.
// The version prevents a late Resume ack from undoing even a timed-out pause.
func (c *Controller) beginPause() (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.session.Pause(); err != nil {
		return nil, err
	}
	c.pausing++
	c.pauseVersion++
	return func() { c.mu.Lock(); c.pausing--; c.mu.Unlock() }, nil
}
func (c *Controller) Pause(ctx context.Context, owner SessionOwner, id string) (err error) {
	if err = c.authorize(owner, id); err != nil {
		return err
	}
	finish, err := c.beginPause()
	if err != nil {
		return err
	}
	var release func()
	// Cancellation is a fallback: native RPC cancellation poisons the helper.
	// Keep the control gate until cancellation and pending-pause cleanup finish.
	defer func() {
		if err != nil {
			c.cancelAction()
		}
		finish()
		if release != nil {
			release()
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, controllerControlGrace)
	defer cancel()
	release, err = lockControllerGate(ctx, c.control)
	if err != nil {
		return err
	}
	if err = c.session.Pause(); err != nil { // Stop may have won while queued.
		return err
	}
	if err = controlResult(ctx, c.backend.Pause(ctx)); err != nil {
		return err
	}
	drain, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return err
	}
	defer drain()
	return nil
}
func (c *Controller) Resume(ctx context.Context, owner SessionOwner, id string) error {
	if err := c.authorize(owner, id); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, controllerControlGrace)
	defer cancel()
	release, err := lockControllerGate(ctx, c.control)
	if err != nil {
		return err
	}
	defer release()
	drain, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return err
	}
	defer drain()
	c.mu.Lock()
	version := c.pauseVersion
	canResume := c.pausing == 0 && c.session.State() == SessionPaused && c.session.Approved()
	c.mu.Unlock()
	if !canResume {
		return errors.New("computer session cannot resume")
	}
	if err = controlResult(ctx, c.backend.Resume(ctx)); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("computer backend cannot resume")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pausing != 0 || c.pauseVersion != version {
		return errors.New("computer resume superseded by pause")
	}
	return c.session.Resume(owner)
}
func (c *Controller) Stop(ctx context.Context, owner SessionOwner, id string) error {
	if err := c.authorize(owner, id); err != nil {
		return err
	}
	if err := c.session.Stop(owner); err != nil {
		return err
	}
	// Stop bypasses action and Pause/Resume gates. Serialize only other Stops:
	// a backend's idempotent no-op ack must not abort an earlier Stop RPC.
	ctx, cancel := context.WithTimeout(ctx, controllerControlGrace)
	defer cancel()
	release, err := lockControllerGate(ctx, c.stopping)
	if err != nil {
		c.cancelAction()
		return err
	}
	defer release()
	defer c.cancelAction() // Notify the live helper before permanent abort.
	return controlResult(ctx, c.backend.Stop(ctx))
}
func controlResult(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
func (c *Controller) Close(ctx context.Context) error {
	_ = c.session.Stop(c.session.Owner())
	c.cancelAction()
	return c.backend.Close(ctx)
}

var _ Service = (*Controller)(nil)
