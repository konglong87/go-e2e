package main

import (
	"context"
	"errors"
	"fmt"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

var errComputerAgentUnauthorized = errors.New("no approved computer session for this conversation")

// computerAgentService does not create or approve sessions. It borrows the UI's
// controller only when a full, non-local owner binding is already present.
// A local preview/acceptance owner (conversation=0) is intentionally invisible.
type computerAgentService struct {
	manager  *computerManager
	lifetime func() context.Context
}

// EnsureComputerSession is called only after the agent tool's normal
// ComputerUse permission gate. It binds an approved model owner lazily, while
// keeping local preview (owner 0/0/0) outside this path.
func (s computerAgentService) EnsureComputerSession(ctx context.Context, owner cu.SessionOwner) (string, error) {
	if s.manager == nil || !validComputerConversationOwner(owner) {
		return "", errComputerAgentUnauthorized
	}
	lifetime := ctx
	if s.lifetime != nil {
		if value := s.lifetime(); value != nil {
			lifetime = value
		}
	}
	snapshot, err := s.manager.startOwnedWithLifetime(ctx, lifetime, ComputerSessionStartInput{Approved: true}, owner)
	if err != nil {
		startupLog("computer ensure failed: " + err.Error())
		return "", err
	}
	return snapshot.ID, nil
}

func (s computerAgentService) controller(ctx context.Context, owner cu.SessionOwner, id string, lookup bool) (*cu.Controller, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.manager == nil || owner.TenantID == 0 || owner.UserID == 0 || owner.SessionID == 0 {
		return nil, errComputerAgentUnauthorized
	}
	s.manager.mu.Lock()
	defer s.manager.mu.Unlock()
	c := s.manager.controller
	if c == nil {
		return nil, errComputerAgentUnauthorized
	}
	session := c.Session()
	if !session.Owns(owner) || !session.Approved() || (!lookup && session.ID() != id) || session.State() == cu.SessionStopped || session.State() == cu.SessionFailed {
		startupLog(fmt.Sprintf("computer controller denied: owner=%+v session_owner=%+v requested_session=%q actual_session=%q approved=%t state=%q lookup=%t", owner, session.Owner(), id, session.ID(), session.Approved(), session.State(), lookup))
		return nil, errComputerAgentUnauthorized
	}
	return c, nil
}
func (s computerAgentService) Lookup(ctx context.Context, owner cu.SessionOwner) (string, error) {
	c, err := s.controller(ctx, owner, "", true)
	if err != nil {
		return "", err
	}
	return c.Session().ID(), nil
}
func (s computerAgentService) Capabilities(ctx context.Context, owner cu.SessionOwner, id string) (cu.Capabilities, error) {
	c, err := s.controller(ctx, owner, id, false)
	if err != nil {
		return cu.Capabilities{}, err
	}
	return c.Capabilities(ctx, owner, id)
}
func (s computerAgentService) Observe(ctx context.Context, owner cu.SessionOwner, r cu.ObserveRequest) (cu.Observation, error) {
	c, err := s.controller(ctx, owner, r.SessionID, false)
	if err != nil {
		return cu.Observation{}, err
	}
	return c.Observe(ctx, owner, r)
}
func (s computerAgentService) Execute(ctx context.Context, owner cu.SessionOwner, a cu.Action) (cu.ActionReceipt, error) {
	c, err := s.controller(ctx, owner, a.SessionID, false)
	if err != nil {
		return cu.ActionReceipt{ActionID: a.ID, SessionID: a.SessionID, Outcome: cu.OutcomeRejected, Verification: cu.VerificationNotChecked}, err
	}
	return c.Execute(ctx, owner, a)
}
func (s computerAgentService) Pause(ctx context.Context, owner cu.SessionOwner, id string) error {
	c, err := s.controller(ctx, owner, id, false)
	if err != nil {
		return err
	}
	return c.Pause(ctx, owner, id)
}
func (s computerAgentService) Resume(ctx context.Context, owner cu.SessionOwner, id string) error {
	c, err := s.controller(ctx, owner, id, false)
	if err != nil {
		return err
	}
	return c.Resume(ctx, owner, id)
}
func (s computerAgentService) Stop(ctx context.Context, owner cu.SessionOwner, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.manager == nil || !validComputerConversationOwner(owner) {
		return errComputerAgentUnauthorized
	}
	s.manager.mu.Lock()
	c := s.manager.controller
	// Revocation clears Approved. The exact old owner may repeat Stop, but must
	// never discover/use that stopped grant or affect a replacement controller.
	if c == nil || c.Session().ID() != id || !c.Session().Owns(owner) {
		s.manager.mu.Unlock()
		return errComputerAgentUnauthorized
	}
	session := c.Session()
	allowed := session.Approved() || session.State() == cu.SessionStopped || session.State() == cu.SessionFailed
	s.manager.mu.Unlock()
	if !allowed {
		return errComputerAgentUnauthorized
	}
	return c.Stop(ctx, owner, id)
}

func (s computerAgentService) ObservationImage(ctx context.Context, owner cu.SessionOwner, id, observationID string) ([]byte, string, error) {
	c, err := s.controller(ctx, owner, id, false)
	if err != nil {
		return nil, "", err
	}
	return c.ObservationImage(ctx, owner, id, observationID)
}
