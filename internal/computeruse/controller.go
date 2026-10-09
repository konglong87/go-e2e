package computeruse

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/konglong87/go-e2e/internal/computerdiag"
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
	session        *ComputerSession
	backend        Backend
	serial         chan struct{}
	control        chan struct{} // Pause/Resume only; Stop must never queue behind them.
	stopping       chan struct{} // Repeat Stop must not abort an earlier Stop acknowledgement.
	mu             sync.Mutex
	cancel         context.CancelFunc
	targetRegistry TargetRegistry
	runner         *Runner
	pausing        int
	pauseVersion   uint64
}

func NewController(s *ComputerSession, b Backend) (*Controller, error) {
	return NewControllerWithRegistry(s, b, nil)
}

func NewControllerWithRegistry(s *ComputerSession, b Backend, registry TargetRegistry) (*Controller, error) {
	return NewControllerWithBudget(s, b, registry, DefaultRunBudget())
}

// NewControllerWithBudget constructs a Controller with an explicit run budget.
// A zero-value budget uses DefaultRunBudget. Hosts that need longer autonomous
// runs (for example complex multi-step desktop tasks) can pass a looser budget;
// MaxUnknownReplays must remain zero.
func NewControllerWithBudget(s *ComputerSession, b Backend, registry TargetRegistry, budget RunBudget) (*Controller, error) {
	if s == nil || b == nil {
		return nil, errors.New("computer session and backend are required")
	}
	if budget == (RunBudget{}) {
		budget = DefaultRunBudget()
	}
	runner, err := NewRunner(budget)
	if err != nil {
		return nil, err
	}
	return &Controller{session: s, backend: b, targetRegistry: registry, runner: runner, serial: make(chan struct{}, 1), control: make(chan struct{}, 1), stopping: make(chan struct{}, 1)}, nil
}
func (c *Controller) Session() *ComputerSession { return c.session }
func (c *Controller) authorize(owner SessionOwner, id string) error {
	if c.session.ID() != id || !c.session.Owns(owner) {
		return errors.New("computer session ownership mismatch")
	}
	return nil
}

func (c *Controller) RunSnapshot() RunSnapshot {
	if c.runner == nil {
		return RunSnapshot{}
	}
	return c.runner.Snapshot()
}

func (c *Controller) failRun() {
	if c.runner != nil {
		_ = c.runner.Transition(Transition{To: RunStateFailed})
	}
}

func (c *Controller) stopRun() {
	if c.runner != nil {
		_ = c.runner.Transition(Transition{To: RunStateStopped})
	}
}

func (c *Controller) ensureBoundRun() error {
	if c.runner == nil || c.runner.State() != RunStateCreated {
		return nil
	}
	return c.runner.Transition(Transition{From: RunStateCreated, To: RunStateBound})
}

func (c *Controller) syncObservedRun() error {
	if err := c.ensureBoundRun(); err != nil {
		return err
	}
	if c.runner != nil && c.runner.State() == RunStateBound {
		return c.runner.Transition(Transition{From: RunStateBound, To: RunStateObserved})
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
func (c *Controller) acquire(ctx context.Context, timeout time.Duration) (context.Context, func(), error) {
	release, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return nil, nil, err
	}
	c.mu.Lock()
	op, cancel := context.WithTimeout(ctx, timeout)
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
	op, release, err := c.acquire(ctx, time.Duration(MaxActionDurationMS)*time.Millisecond)
	if err != nil {
		return Observation{}, err
	}
	defer release()
	if err = c.session.CanObserve(); err != nil {
		return Observation{}, err
	}
	if err = c.ensureBoundRun(); err != nil {
		return Observation{}, err
	}
	started := time.Now()
	o, err := c.backend.Observe(op, r)
	if err != nil {
		var coded interface{ Code() string }
		if errors.As(err, &coded) && transientFailureCode(coded.Code()) {
			// Focus change or permission state is transient: pause and reopen
			// the observation phase so a fresh Observe can re-activate the
			// target and resume. Do not permanently fail the run.
			c.pauseForRecovery()
		} else {
			_ = c.session.Pause()
			c.failRun()
		}
		return Observation{}, observeFailure(err)
	}
	if err = c.session.SetObservation(o); err != nil {
		_ = c.session.Pause()
		c.failRun()
		return Observation{}, err
	}
	if c.runner != nil {
		if err = c.runner.Consume(Consumption{Kind: BudgetObserve, Duration: time.Since(started)}); err != nil {
			c.failRun()
			return Observation{}, err
		}
		if c.runner.State() == RunStateBound || c.runner.State() == RunStateVerifying {
			from := c.runner.State()
			if err = c.runner.Transition(Transition{From: from, To: RunStateObserved}); err != nil {
				c.failRun()
				return Observation{}, err
			}
		}
	}
	return o, nil
}

// LaunchTarget starts a registered target through the trusted backend. The
// model-visible TargetID is resolved before any provider key crosses the
// backend boundary.
func (c *Controller) LaunchTarget(ctx context.Context, owner SessionOwner, id string, targetID TargetID) (LaunchReceipt, error) {
	if err := c.authorize(owner, id); err != nil {
		return LaunchReceipt{TargetID: targetID, Outcome: OutcomeRejected, ErrorCode: ErrorCodeInactive, CompletedAt: time.Now()}, err
	}
	release, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return LaunchReceipt{TargetID: targetID, Outcome: OutcomeRejected, ErrorCode: ErrorCodeTimeout, CompletedAt: time.Now()}, err
	}
	defer release()
	started := time.Now()
	receipt := LaunchReceipt{TargetID: targetID, Outcome: OutcomeRejected, CompletedAt: time.Now()}
	if c.targetRegistry == nil {
		receipt.ErrorCode = ErrorCodeUnsupportedTarget
		return receipt, errors.New("computer target registry is unavailable")
	}
	target, err := c.targetRegistry.Resolve(targetID)
	if err != nil {
		receipt.ErrorCode = ErrorCodeUnsupportedTarget
		return receipt, errors.New("computer target is not registered")
	}
	receipt.DisplayName = target.DisplayName
	if !c.session.Approved() || c.session.State() == SessionStopped || c.session.State() == SessionFailed {
		receipt.ErrorCode = ErrorCodeInactive
		return receipt, errors.New("computer session is not ready")
	}
	launcher, ok := c.backend.(BackendLauncher)
	if !ok {
		receipt.ErrorCode = ErrorCodeLaunchFailed
		return receipt, errors.New("computer target launcher is unavailable")
	}
	receipt, err = launcher.LaunchApp(ctx, id, target.Launch.ProviderKey)
	receipt.TargetID = target.ID
	receipt.DisplayName = target.DisplayName
	if receipt.ErrorCode != "" {
		receipt.ErrorCode = PublicErrorCode(receipt.ErrorCode)
	}
	if receipt.Outcome == OutcomeExecuted {
		if c.runner != nil {
			if budgetErr := c.runner.Consume(Consumption{Kind: BudgetLaunch, Duration: time.Since(started)}); budgetErr != nil {
				c.failRun()
				receipt.Outcome = OutcomeRejected
				receipt.ErrorCode = ErrorCodeTimeout
				return receipt, budgetErr
			}
		}
		if target.Window.BundleID != "" && receipt.Window.BundleID != target.Window.BundleID {
			receipt.Outcome = OutcomeRejected
			receipt.ErrorCode = ErrorCodeTargetWindowMismatch
			err = errors.New("computer target window does not match registry policy")
		}
		if target.Window.RequireVisible && !receipt.Window.IsVisible {
			computerdiag.Append(computerdiag.GoMismatchPath, map[string]any{
				"layer": "go_controller", "phase": "launch_require_visible", "target_id": target.ID,
				"window_id": receipt.Window.ID, "owner_pid": receipt.Window.OwnerPID, "bundle_id": receipt.Window.BundleID,
				"is_visible": receipt.Window.IsVisible, "is_frontmost": receipt.Window.IsFrontmost, "error_code": ErrorCodeTargetWindowMismatch,
			})
			receipt.Outcome = OutcomeRejected
			receipt.ErrorCode = ErrorCodeTargetWindowMismatch
			err = errors.New("computer target window is not visible")
		}
		if target.Window.RequireFrontmost && !receipt.Window.IsFrontmost {
			receipt.Outcome = OutcomeRejected
			receipt.ErrorCode = ErrorCodeTargetWindowMismatch
			err = errors.New("computer target window is not frontmost")
		}
		if err == nil && c.runner != nil {
			from := c.runner.State()
			if from == RunStateCreated || from == RunStateObserved {
				if err = c.runner.Transition(Transition{From: from, To: RunStateBound}); err != nil {
					c.failRun()
					receipt.Outcome = OutcomeRejected
					receipt.ErrorCode = ErrorCodeTargetWindowMismatch
				}
			}
		}
	}
	return receipt, err
}

// LaunchApp is retained for older adapters. New callers must use LaunchTarget.
func (c *Controller) LaunchApp(ctx context.Context, owner SessionOwner, id, app string) (LaunchReceipt, error) {
	if err := c.authorize(owner, id); err != nil {
		return LaunchReceipt{}, err
	}
	release, err := lockControllerGate(ctx, c.serial)
	if err != nil {
		return LaunchReceipt{}, err
	}
	defer release()
	if !c.session.Approved() || c.session.State() == SessionStopped || c.session.State() == SessionFailed {
		return LaunchReceipt{Application: ComputerApplication(app), Outcome: OutcomeRejected, ErrorCode: ErrorCodeInactive, CompletedAt: time.Now()}, errors.New("computer session is not ready")
	}
	launcher, ok := c.backend.(BackendApplicationLauncher)
	if !ok {
		return LaunchReceipt{Application: ComputerApplication(app), Outcome: OutcomeRejected, ErrorCode: ErrorCodeLaunchFailed, CompletedAt: time.Now()}, errors.New("computer application launcher is unavailable")
	}
	receipt, err := launcher.LaunchApp(ctx, id, app)
	if receipt.ErrorCode != "" {
		receipt.ErrorCode = PublicErrorCode(receipt.ErrorCode)
	}
	return receipt, err
}

type observeFailureError struct {
	code string
}

func (e observeFailureError) Error() string {
	return "computer observation failed: " + e.code
}

func (e observeFailureError) Code() string {
	return e.code
}

func observeFailure(err error) error {
	var coded interface{ Code() string }
	if errors.As(err, &coded) {
		if code := PublicErrorCode(coded.Code()); code != ErrorCodeActionFailed {
			return observeFailureError{code: code}
		}
	}
	return observeFailureError{code: ErrorCodeActionFailed}
}

// transientFailureCode reports whether a backend error code represents a
// read-only or pre-dispatch recoverable condition rather than helper
// corruption. A transient failure pauses the run and reopens the observation
// phase via ResetToObserved, instead of permanently failing the run. The
// recovery path is always "re-observe the real state and let the model
// decide"; it never replays the interrupted input.
func transientFailureCode(code string) bool {
	switch code {
	case ErrorCodeFocusChanged, ErrorCodePermissionRequired, ErrorCodeScreenshotFailed,
		ErrorCodeTargetWindowMismatch, ErrorCodeUnsupportedDisplay:
		return true
	}
	return false
}

// pauseForRecovery pauses the session and reopens the observation phase for a
// pre-dispatch failure. User Resume is still required before following input.
func (c *Controller) pauseForRecovery() {
	_ = c.session.Pause()
	if c.runner != nil {
		_ = c.runner.ResetToObserved()
	}
}

func (c *Controller) Execute(ctx context.Context, owner SessionOwner, a Action) (ActionReceipt, error) {
	if err := c.authorize(owner, a.SessionID); err != nil {
		return ActionReceipt{}, err
	}
	op, release, err := c.acquire(ctx, ActionExecutionTimeout(a, time.Duration(MaxActionDurationMS)*time.Millisecond))
	if err != nil {
		return ActionReceipt{}, err
	}
	defer release()
	before, _ := c.session.CurrentObservation()
	// Once a window-scoped observation exists, omission of window_id is filled
	// from trusted observation state. The backend still validates the exact
	// Window Server identity before dispatch; model input cannot silently fall
	// back to display-level injection.
	if before.WindowID != "" && a.WindowID == "" {
		a.WindowID = before.WindowID
	}
	if err = c.session.BeginAction(a); err != nil {
		return ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: OutcomeRejected, Verification: VerificationNotChecked, RedactedActionSummary: a.RedactedSummary(), ErrorCode: "action_rejected", CompletedAt: time.Now()}, err
	}
	if err = c.syncObservedRun(); err != nil {
		return ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: OutcomeRejected, Verification: VerificationNotChecked, RedactedActionSummary: a.RedactedSummary(), ErrorCode: ErrorCodeInactive, CompletedAt: time.Now()}, err
	}
	if c.runner != nil {
		kind := BudgetInput
		if a.Kind == ActionWait {
			kind = BudgetWait
		}
		// Reserve the count before dispatch. An over-budget action must never
		// reach the native host; an unknown dispatch is still never replayed.
		if err = c.runner.Consume(Consumption{Kind: kind}); err != nil {
			c.failRun()
			return ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: OutcomeRejected, Verification: VerificationNotChecked, RedactedActionSummary: a.RedactedSummary(), ErrorCode: ErrorCodeTimeout, CompletedAt: time.Now()}, err
		}
		if err = c.runner.Transition(Transition{From: RunStateObserved, To: RunStateExecuting}); err != nil {
			c.failRun()
			return ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: OutcomeRejected, Verification: VerificationNotChecked, RedactedActionSummary: a.RedactedSummary(), ErrorCode: ErrorCodeInactive, CompletedAt: time.Now()}, err
		}
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
		c.failRun()
		return receipt, err
	}
	if c.runner != nil {
		cooperativePause := c.session.State() == SessionPaused
		if cooperativePause {
			// Pause owns the recovery boundary and resets the runner after the
			// interrupted receipt has drained.
		} else if receipt.Outcome == OutcomeUnknown || backendErr != nil {
			if receipt.Outcome != OutcomeUnknown && transientFailureCode(receipt.ErrorCode) {
				// Focus change or permission state is transient: pause and
				// reopen the observation phase so a fresh Observe can resume.
				// The interrupted input is never replayed.
				c.pauseForRecovery()
			} else {
				c.failRun()
			}
		} else if err = c.runner.Transition(Transition{From: RunStateExecuting, To: RunStateVerifying}); err != nil {
			c.failRun()
		} else if receipt.AfterObservationID != "" && receipt.After != nil {
			_ = c.runner.Transition(Transition{From: RunStateVerifying, To: RunStateObserved})
		} else if receipt.Outcome != OutcomeExecuted || receipt.DispatchState != DispatchComplete {
			c.failRun()
		}
	}
	if backendErr != nil {
		return receipt, errors.New("computer backend failed; inspect receipt before continuing")
	}
	if (receipt.AfterObservationID == "" || receipt.After == nil) && receipt.ErrorCode == ErrorCodePermissionRequired {
		// Revoked permission requires explicit user recovery, never auto-resume.
		c.pauseForRecovery()
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
	if c.runner != nil {
		if resetErr := c.runner.ResetToObserved(); resetErr != nil {
			c.failRun()
			return resetErr
		}
	}
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
	canResume := c.pausing == 0 && c.session.State() == SessionPaused && c.session.Approved() && !c.session.InputUncertain()
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
	c.stopRun()
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
	// Give the live helper a bounded chance to release held input before
	// cancellation poisons its RPC transport. Stop revokes authority first.
	stopErr := c.Stop(ctx, c.session.Owner(), c.session.ID())
	return errors.Join(stopErr, c.backend.Close(ctx))
}

var _ Service = (*Controller)(nil)
