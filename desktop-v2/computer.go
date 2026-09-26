package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	macbackend "github.com/konglong87/go-e2e/internal/computerbackend/macos"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const (
	computerHelperName     = "computer-helper-macos"
	computerHelperEnv      = "GO_E2E_COMPUTER_HELPER"
	computerRequestTimeout = 10 * time.Second
	// This identity is local to the Wails-only control surface. It is NOT a
	// tenant runtime credential and must never be injected into a model Query.
	localComputerTenantID uint64 = 1
	localComputerUserID   uint64 = 1
)

type ComputerSessionStartInput struct {
	Approved bool `json:"approved"`
}
type ComputerCapabilitiesDTO struct {
	Capabilities cu.Capabilities `json:"capabilities"`
	Available    bool            `json:"available"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}
type ComputerObservationDTO struct {
	Observation cu.Observation `json:"observation"`
	ImageData   string         `json:"image_data,omitempty"`
	MediaType   string         `json:"media_type,omitempty"`
}
type ComputerSessionDTO struct {
	ID           string            `json:"session_id"`
	State        cu.SessionState   `json:"state"`
	Capabilities cu.Capabilities   `json:"capabilities"`
	Observation  *cu.Observation   `json:"observation,omitempty"`
	LastReceipt  *cu.ActionReceipt `json:"last_receipt,omitempty"`
}

type computerBackendFactory func(context.Context) (cu.Backend, error)

// Platform construction is injected; control/snapshot logic depends only on
// the domain controller. Windows will supply a factory, not a second manager.
type computerManager struct {
	mu         sync.Mutex
	backend    cu.Backend
	controller *cu.Controller
	owner      cu.SessionOwner
	factory    computerBackendFactory
}

func newComputerManager() *computerManager {
	return &computerManager{owner: cu.SessionOwner{TenantID: localComputerTenantID, UserID: localComputerUserID}, factory: newComputerBackend}
}
func newComputerBackend(ctx context.Context) (cu.Backend, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("Computer Use backend is unavailable on this platform")
	}
	path, err := locateComputerHelper()
	if err != nil {
		return nil, err
	}
	return macbackend.New(ctx, macbackend.Config{
		HelperPath:             path,
		RequestTimeout:         computerRequestTimeout,
		RequestHostPermissions: macbackend.RequestHostPermissions,
	})
}
func locateComputerHelper() (string, error) {
	if value := strings.TrimSpace(os.Getenv(computerHelperEnv)); value != "" {
		if !filepath.IsAbs(value) {
			return "", errors.New("computer helper path must be absolute")
		}
		return value, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(filepath.Dir(executable), "..", "Helpers", "ComputerHelper.app", "Contents", "MacOS", computerHelperName),
		filepath.Join(filepath.Dir(executable), "..", "Helpers", computerHelperName),
	}
	for _, path := range candidates {
		if stat, statErr := os.Stat(path); statErr == nil && !stat.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("computer helper is not bundled")
}
func (m *computerManager) ensureBackendLocked(ctx context.Context) (cu.Backend, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.backend != nil {
		return m.backend, nil
	}
	b, err := m.factory(ctx)
	if err != nil {
		return nil, err
	}
	m.backend = b
	return b, nil
}
func (m *computerManager) capabilities(ctx context.Context) (cu.Capabilities, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := m.ensureBackendLocked(ctx)
	if err != nil {
		return cu.Capabilities{}, err
	}
	return b.Capabilities(ctx)
}
func (m *computerManager) start(ctx context.Context, in ComputerSessionStartInput) (ComputerSessionDTO, error) {
	return m.startWithLifetime(ctx, ctx, in)
}

// The host owns the helper lifetime; approval belongs to the live caller.
func (m *computerManager) startWithLifetime(ctx, lifetime context.Context, in ComputerSessionStartInput) (ComputerSessionDTO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ComputerSessionDTO{}, err
	}
	if err := lifetime.Err(); err != nil {
		return ComputerSessionDTO{}, err
	}
	if m.controller != nil {
		s := m.controller.Session()
		if s.State() != cu.SessionStopped && s.State() != cu.SessionFailed {
			if in.Approved && s.State() == cu.SessionPendingApproval {
				if err := s.Approve(m.owner); err != nil {
					return ComputerSessionDTO{}, err
				}
			}
			return computerSnapshot(s), nil
		}
		// A stopped helper must not be reused for a newly approved session.
		if err := m.controller.Close(ctx); err != nil {
			return ComputerSessionDTO{}, err
		}
		m.backend = nil
		m.controller = nil
	}
	b, err := m.ensureBackendLocked(lifetime)
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	caps, err := b.Capabilities(ctx)
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	if err := ctx.Err(); err != nil {
		return ComputerSessionDTO{}, err
	}
	s, err := cu.NewComputerSession(cu.SessionOptions{Owner: m.owner, Capabilities: caps})
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	c, err := cu.NewController(s, b)
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	m.controller = c
	if in.Approved {
		if err := s.Approve(m.owner); err != nil {
			return ComputerSessionDTO{}, err
		}
	}
	return computerSnapshot(s), nil
}
func (m *computerManager) active(id string) (*cu.Controller, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.controller == nil || m.controller.Session().ID() != id {
		return nil, errors.New("computer session not found")
	}
	return m.controller, nil
}
func computerSnapshot(s *cu.ComputerSession) ComputerSessionDTO {
	result := ComputerSessionDTO{ID: s.ID(), State: s.State(), Capabilities: s.Capabilities()}
	if o, ok := s.CurrentObservation(); ok {
		result.Observation = &o
	}
	if r, ok := s.LastReceipt(); ok {
		result.LastReceipt = &r
	}
	return result
}
func (m *computerManager) observe(ctx context.Context, id string) (ComputerObservationDTO, error) {
	c, err := m.active(id)
	if err != nil {
		return ComputerObservationDTO{}, err
	}
	o, err := c.Observe(ctx, m.owner, cu.ObserveRequest{SessionID: id})
	if err != nil {
		return ComputerObservationDTO{}, err
	}
	data, mediaType, err := c.ObservationImage(ctx, m.owner, id, o.ID)
	if err != nil {
		_ = c.Pause(ctx, m.owner, id)
		return ComputerObservationDTO{}, err
	}
	return ComputerObservationDTO{Observation: o, ImageData: base64.StdEncoding.EncodeToString(data), MediaType: mediaType}, nil
}
func (m *computerManager) control(ctx context.Context, id string, kind cu.ActionKind) (ComputerSessionDTO, error) {
	c, err := m.active(id)
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	switch kind {
	case cu.ActionPause:
		err = c.Pause(ctx, m.owner, id)
	case cu.ActionResume:
		err = c.Resume(ctx, m.owner, id)
	case cu.ActionStop:
		err = c.Stop(ctx, m.owner, id)
	default:
		err = errors.New("unsupported computer control")
	}
	return computerSnapshot(c.Session()), err
}
func (m *computerManager) close(ctx context.Context) error {
	m.mu.Lock()
	c, b := m.controller, m.backend
	m.controller = nil
	m.backend = nil
	m.mu.Unlock()
	if c != nil {
		return c.Close(ctx)
	}
	if b != nil {
		return b.Close(ctx)
	}
	return nil
}
func (a *app) GetComputerCapabilities() ComputerCapabilitiesDTO {
	caps, err := a.computer().capabilities(a.windowContext())
	if err != nil {
		return ComputerCapabilitiesDTO{ErrorCode: "capability_unavailable", ErrorMessage: err.Error()}
	}
	return ComputerCapabilitiesDTO{Capabilities: caps, Available: true}
}
func (a *app) StartComputerSession(in ComputerSessionStartInput) (ComputerSessionDTO, error) {
	return a.computer().start(a.windowContext(), in)
}
func (a *app) ObserveComputerSession(id string) (ComputerObservationDTO, error) {
	return a.computer().observe(a.windowContext(), id)
}
func (a *app) PauseComputerSession(id string) (ComputerSessionDTO, error) {
	return a.computer().control(a.windowContext(), id, cu.ActionPause)
}
func (a *app) ResumeComputerSession(id string) (ComputerSessionDTO, error) {
	return a.computer().control(a.windowContext(), id, cu.ActionResume)
}
func (a *app) StopComputerSession(id string) (ComputerSessionDTO, error) {
	return a.computer().control(a.windowContext(), id, cu.ActionStop)
}
func (a *app) GetComputerActionReceipt(id, actionID string) (cu.ActionReceipt, error) {
	c, err := a.computer().active(id)
	if err != nil {
		return cu.ActionReceipt{}, err
	}
	r, ok := c.Session().Receipt(actionID)
	if !ok {
		return cu.ActionReceipt{}, errors.New("computer receipt not found")
	}
	return r, nil
}
func (a *app) computer() *computerManager {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.computerManager == nil {
		a.computerManager = newComputerManager()
	}
	return a.computerManager
}
