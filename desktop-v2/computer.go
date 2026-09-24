package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	runtimepkg "runtime"
	"strings"
	"sync"
	"time"

	macbackend "github.com/konglong87/go-e2e/internal/computerbackend/macos"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
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
	ID           string            `json:"id"`
	State        cu.SessionState   `json:"state"`
	Capabilities cu.Capabilities   `json:"capabilities"`
	Observation  *cu.Observation   `json:"observation,omitempty"`
	LastReceipt  *cu.ActionReceipt `json:"last_receipt,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type computerManager struct {
	mu      sync.Mutex
	backend *macbackend.Backend
	session *cu.ComputerSession
	owner   cu.SessionOwner
}

func newComputerManager() *computerManager {
	return &computerManager{owner: cu.SessionOwner{TenantID: 1, UserID: 1}}
}

func (m *computerManager) helperPath() (string, error) {
	if value := strings.TrimSpace(os.Getenv("GO_E2E_COMPUTER_HELPER")); value != "" {
		return value, nil
	}
	executable, err := os.Executable()
	if err == nil {
		candidates := []string{
			filepath.Join(filepath.Dir(executable), "..", "Helpers", "computer-helper-macos"),
			filepath.Join(filepath.Dir(executable), "computer-helper-macos"),
		}
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
		}
	}
	if runtimepkg.GOOS != "darwin" {
		return "", errors.New("macOS Host Computer Use is only available on macOS")
	}
	return "", errors.New("computer helper is not bundled; set GO_E2E_COMPUTER_HELPER")
}

func (m *computerManager) ensureBackend(ctx context.Context) (*macbackend.Backend, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backend != nil {
		return m.backend, nil
	}
	path, err := m.helperPath()
	if err != nil {
		return nil, err
	}
	backend, err := macbackend.New(ctx, macbackend.Config{HelperPath: path, RequestTimeout: 10 * time.Second})
	if err != nil {
		return nil, err
	}
	m.backend = backend
	return backend, nil
}

func (m *computerManager) capabilities(ctx context.Context) (cu.Capabilities, error) {
	backend, err := m.ensureBackend(ctx)
	if err != nil {
		return cu.Capabilities{}, err
	}
	return backend.Capabilities(ctx)
}

func (m *computerManager) start(ctx context.Context, input ComputerSessionStartInput) (ComputerSessionDTO, error) {
	caps, err := m.capabilities(ctx)
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil && m.session.State() != cu.SessionStopped {
		return m.snapshotLocked(), nil
	}
	session, err := cu.NewComputerSession(cu.SessionOptions{Owner: m.owner, Capabilities: caps, RequireApproval: true, MaxActions: 30})
	if err != nil {
		return ComputerSessionDTO{}, err
	}
	m.session = session
	if input.Approved {
		if err := m.session.Approve(m.owner); err != nil {
			return ComputerSessionDTO{}, err
		}
	}
	return m.snapshotLocked(), nil
}

func (m *computerManager) observe(ctx context.Context, owner cu.SessionOwner, request cu.ObserveRequest) (cu.Observation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(owner, request.SessionID); err != nil {
		return cu.Observation{}, err
	}
	if m.session.State() != cu.SessionReady {
		return cu.Observation{}, fmt.Errorf("computer session is not ready: %s", m.session.State())
	}
	observation, err := m.backend.Observe(ctx, request)
	if err != nil {
		return cu.Observation{}, err
	}
	if err := m.session.SetObservation(observation); err != nil {
		return cu.Observation{}, err
	}
	return observation, nil
}

func (m *computerManager) execute(ctx context.Context, owner cu.SessionOwner, action cu.Action) (cu.ActionReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(owner, action.SessionID); err != nil {
		return cu.ActionReceipt{}, err
	}
	if err := m.session.ValidateAction(action); err != nil {
		return cu.ActionReceipt{ActionID: action.ID, SessionID: action.SessionID, Outcome: cu.OutcomeRejected, Verification: cu.VerificationNotChecked, RedactedActionSummary: action.RedactedSummary(), ErrorMessage: err.Error()}, err
	}
	receipt, err := m.backend.Execute(ctx, action)
	if recordErr := m.session.RecordReceipt(receipt); recordErr != nil && err == nil {
		err = recordErr
	}
	return receipt, err
}

func (m *computerManager) pause(ctx context.Context, owner cu.SessionOwner, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(owner, id); err != nil {
		return err
	}
	if err := m.session.Pause(); err != nil {
		return err
	}
	return m.backend.Pause(ctx)
}

func (m *computerManager) resume(ctx context.Context, owner cu.SessionOwner, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(owner, id); err != nil {
		return err
	}
	if err := m.session.Resume(owner); err != nil {
		return err
	}
	return m.backend.Resume(ctx)
}

func (m *computerManager) stop(ctx context.Context, owner cu.SessionOwner, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(owner, id); err != nil {
		return err
	}
	_ = m.backend.Stop(ctx)
	return m.session.Stop(owner)
}

func (m *computerManager) close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backend == nil {
		return nil
	}
	err := m.backend.Close(ctx)
	m.backend = nil
	return err
}

func (m *computerManager) checkOwnerLocked(owner cu.SessionOwner, id string) error {
	if m.session == nil || m.session.ID() != id {
		return errors.New("computer session not found")
	}
	if !m.session.Owns(owner) {
		return errors.New("computer session ownership mismatch")
	}
	return nil
}

func (m *computerManager) snapshotLocked() ComputerSessionDTO {
	snapshot := ComputerSessionDTO{Capabilities: m.session.Capabilities(), ID: m.session.ID(), State: m.session.State()}
	if observation, ok := m.session.CurrentObservation(); ok {
		snapshot.Observation = &observation
	}
	if receipt, ok := m.session.LastReceipt(); ok {
		snapshot.LastReceipt = &receipt
	}
	return snapshot
}

func (a *app) GetComputerCapabilities() ComputerCapabilitiesDTO {
	caps, err := a.computer().capabilities(a.windowContext())
	if err != nil {
		return ComputerCapabilitiesDTO{Available: false, ErrorCode: "capability_unavailable", ErrorMessage: err.Error()}
	}
	return ComputerCapabilitiesDTO{Capabilities: caps, Available: true}
}

func (a *app) StartComputerSession(input ComputerSessionStartInput) (ComputerSessionDTO, error) {
	return a.computer().start(a.windowContext(), input)
}

func (a *app) ObserveComputerSession(sessionID string) (ComputerObservationDTO, error) {
	observation, err := a.computer().observe(a.windowContext(), a.computer().owner, cu.ObserveRequest{SessionID: sessionID})
	if err != nil {
		return ComputerObservationDTO{}, err
	}
	image, mediaType, err := a.computer().backend.ObservationImage(a.windowContext(), observation.ID)
	if err != nil {
		return ComputerObservationDTO{}, err
	}
	return ComputerObservationDTO{Observation: observation, ImageData: base64.StdEncoding.EncodeToString(image), MediaType: mediaType}, nil
}

func (a *app) PauseComputerSession(sessionID string) error {
	return a.computer().pause(a.windowContext(), a.computer().owner, sessionID)
}
func (a *app) ResumeComputerSession(sessionID string) error {
	return a.computer().resume(a.windowContext(), a.computer().owner, sessionID)
}
func (a *app) StopComputerSession(sessionID string) error {
	return a.computer().stop(a.windowContext(), a.computer().owner, sessionID)
}
func (a *app) GetComputerActionReceipt(sessionID, actionID string) (cu.ActionReceipt, error) {
	m := a.computer()
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkOwnerLocked(m.owner, sessionID); err != nil {
		return cu.ActionReceipt{}, err
	}
	receipt, ok := m.session.LastReceipt()
	if !ok || receipt.ActionID != actionID {
		return cu.ActionReceipt{}, errors.New("computer action receipt not found")
	}
	return receipt, nil
}

func (a *app) computer() *computerManager {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.computerManager == nil {
		a.computerManager = newComputerManager()
	}
	return a.computerManager
}
