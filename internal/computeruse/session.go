package computeruse

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionState string

const (
	SessionPendingApproval  SessionState = "pending_approval"
	SessionReady            SessionState = "ready"
	SessionPaused           SessionState = "paused"
	SessionNeedsObservation SessionState = "needs_observation"
	SessionStopped          SessionState = "stopped"
	SessionFailed           SessionState = "failed"
)

type SessionOwner struct {
	TenantID uint64 `json:"tenant_id"`
	UserID   uint64 `json:"user_id"`
}

type SessionOptions struct {
	ID              string
	Owner           SessionOwner
	Capabilities    Capabilities
	RequireApproval bool
	MaxActions      int
	ObservationTTL  time.Duration
	Now             func() time.Time
}

type ComputerSession struct {
	mu sync.RWMutex

	id             string
	owner          SessionOwner
	capabilities   Capabilities
	state          SessionState
	approved       bool
	maxActions     int
	actionCount    int
	observationTTL time.Duration
	now            func() time.Time
	observation    Observation
	hasObservation bool
	lastReceipt    *ActionReceipt
}

func NewComputerSession(options SessionOptions) (*ComputerSession, error) {
	if options.ID == "" {
		options.ID = uuid.NewString()
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.MaxActions <= 0 {
		options.MaxActions = 30
	}
	if options.ObservationTTL <= 0 {
		options.ObservationTTL = 30 * time.Second
	}
	if err := options.Capabilities.Validate(); err != nil {
		return nil, err
	}
	state := SessionReady
	approved := true
	if options.RequireApproval || options.Capabilities.PermissionState != PermissionApproved {
		state = SessionPendingApproval
		approved = false
	}
	return &ComputerSession{
		id:             options.ID,
		owner:          options.Owner,
		capabilities:   options.Capabilities,
		state:          state,
		approved:       approved,
		maxActions:     options.MaxActions,
		observationTTL: options.ObservationTTL,
		now:            options.Now,
	}, nil
}

func (s *ComputerSession) ID() string          { s.mu.RLock(); defer s.mu.RUnlock(); return s.id }
func (s *ComputerSession) Owner() SessionOwner { s.mu.RLock(); defer s.mu.RUnlock(); return s.owner }
func (s *ComputerSession) State() SessionState { s.mu.RLock(); defer s.mu.RUnlock(); return s.state }
func (s *ComputerSession) Capabilities() Capabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities
}
func (s *ComputerSession) ActionCount() int { s.mu.RLock(); defer s.mu.RUnlock(); return s.actionCount }
func (s *ComputerSession) Approved() bool   { s.mu.RLock(); defer s.mu.RUnlock(); return s.approved }

func (s *ComputerSession) Owns(owner SessionOwner) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.owner == owner
}

func (s *ComputerSession) Approve(owner SessionOwner) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != owner {
		return errors.New("computer session ownership mismatch")
	}
	if s.state != SessionPendingApproval {
		return fmt.Errorf("cannot approve session in state %q", s.state)
	}
	if !s.capabilities.Ready() {
		return errors.New("computer backend is not ready")
	}
	s.approved = true
	s.state = SessionReady
	return nil
}

func (s *ComputerSession) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != SessionReady {
		return fmt.Errorf("cannot pause session in state %q", s.state)
	}
	s.state = SessionPaused
	return nil
}

func (s *ComputerSession) Resume(owner SessionOwner) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != owner {
		return errors.New("computer session ownership mismatch")
	}
	if s.state != SessionPaused {
		return fmt.Errorf("cannot resume session in state %q", s.state)
	}
	s.state = SessionReady
	return nil
}

func (s *ComputerSession) Stop(owner SessionOwner) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != owner {
		return errors.New("computer session ownership mismatch")
	}
	if s.state == SessionStopped {
		return nil
	}
	s.state = SessionStopped
	s.approved = false
	return nil
}

func (s *ComputerSession) SetObservation(observation Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if observation.SessionID != s.id {
		return errors.New("observation session mismatch")
	}
	if observation.ID == "" {
		return errors.New("observation id is required")
	}
	if observation.Width <= 0 || observation.Height <= 0 {
		return errors.New("observation dimensions must be positive")
	}
	if !observation.Capabilities.Ready() {
		return errors.New("observation backend is not ready")
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = s.now()
	}
	if observation.ExpiresAt.IsZero() {
		observation.ExpiresAt = observation.ObservedAt.Add(s.observationTTL)
	}
	s.observation = observation
	s.hasObservation = true
	if s.state == SessionNeedsObservation {
		s.state = SessionReady
	}
	return nil
}

func (s *ComputerSession) CurrentObservation() (Observation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.observation, s.hasObservation
}

func (s *ComputerSession) ValidateAction(action Action) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if action.SessionID != s.id {
		return errors.New("action session mismatch")
	}
	if s.state != SessionReady {
		return fmt.Errorf("session is not ready: %s", s.state)
	}
	if s.actionCount >= s.maxActions {
		return errors.New("computer action budget exceeded")
	}
	if !s.hasObservation && action.Kind.IsInput() {
		return errors.New("input action requires an observation")
	}
	if action.Kind.IsInput() {
		return action.Validate(s.now(), s.observation)
	}
	return action.Validate(s.now(), s.observation)
}

func (s *ComputerSession) RecordReceipt(receipt ActionReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if receipt.SessionID != s.id {
		return errors.New("receipt session mismatch")
	}
	if !receipt.IsTerminal() {
		return errors.New("receipt outcome is not terminal")
	}
	s.lastReceipt = &receipt
	s.actionCount++
	switch receipt.Outcome {
	case OutcomeUnknown:
		s.state = SessionNeedsObservation
	case OutcomeFailed:
		s.state = SessionFailed
	case OutcomeRejected:
		if s.state == SessionReady {
			s.state = SessionReady
		}
	}
	return nil
}

func (s *ComputerSession) LastReceipt() (ActionReceipt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastReceipt == nil {
		return ActionReceipt{}, false
	}
	return *s.lastReceipt, true
}
