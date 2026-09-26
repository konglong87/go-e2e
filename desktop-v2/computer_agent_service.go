package main

import (
	"context"
	"errors"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

var errComputerAgentUnauthorized = errors.New("no approved computer session for this conversation")

// computerAgentService does not create or approve sessions. It borrows the UI's
// controller only when a full, non-local owner binding is already present.
// A local preview/acceptance owner (conversation=0) is intentionally invisible.
type computerAgentService struct{ manager *computerManager }

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
	c, err := s.controller(ctx, owner, id, false)
	if err != nil {
		return err
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
