package computeruse

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ImageReader transfers ephemeral PNG bytes without putting them in the domain
// observation, event, or transcript. Authorization is enforced by Controller.
type ImageReader interface {
	ObservationImage(context.Context, string) ([]byte, string, error)
}

// Controller is one approved host session's execution authority. Agent and UI
// adapters must share this instance; constructing a Tool never grants approval.
type Controller struct {
	session *ComputerSession
	backend Backend
	serial  chan struct{}
	mu      sync.Mutex
	cancel  context.CancelFunc
}

func NewController(s *ComputerSession, b Backend) (*Controller, error) {
	if s == nil || b == nil {
		return nil, errors.New("computer session and backend are required")
	}
	return &Controller{session: s, backend: b, serial: make(chan struct{}, 1)}, nil
}
func (c *Controller) Session() *ComputerSession { return c.session }
func (c *Controller) authorize(owner SessionOwner, id string) error {
	if c.session.ID() != id || !c.session.Owns(owner) {
		return errors.New("computer session ownership mismatch")
	}
	return nil
}
func (c *Controller) acquire(ctx context.Context) (context.Context, func(), error) {
	select {
	case c.serial <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-c.serial
		return nil, nil, err
	}
	c.mu.Lock()
	op, cancel := context.WithTimeout(ctx, time.Duration(MaxActionDurationMS)*time.Millisecond)
	c.cancel = cancel
	c.mu.Unlock()
	return op, func() { cancel(); c.mu.Lock(); c.cancel = nil; c.mu.Unlock(); <-c.serial }, nil
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
	return c.session.Capabilities(), nil
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
		return Observation{}, errors.New("computer capture failed; session paused")
	}
	if err = c.session.SetObservation(o); err != nil {
		_ = c.session.Pause()
		return Observation{}, err
	}
	return o, nil
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
func (c *Controller) Pause(ctx context.Context, owner SessionOwner, id string) error {
	if err := c.authorize(owner, id); err != nil {
		return err
	}
	if err := c.session.Pause(); err != nil {
		return err
	}
	c.cancelAction()
	return c.backend.Pause(ctx)
}
func (c *Controller) Resume(ctx context.Context, owner SessionOwner, id string) error {
	if err := c.authorize(owner, id); err != nil {
		return err
	}
	if c.session.State() != SessionPaused || !c.session.Approved() {
		return errors.New("computer session cannot resume")
	}
	if err := c.backend.Resume(ctx); err != nil {
		return errors.New("computer backend cannot resume")
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
	c.cancelAction()
	return c.backend.Stop(ctx)
}
func (c *Controller) Close(ctx context.Context) error {
	_ = c.session.Stop(c.session.Owner())
	c.cancelAction()
	return c.backend.Close(ctx)
}

var _ Service = (*Controller)(nil)
