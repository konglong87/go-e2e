package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

func testComputerManager() (*computerManager, *int) {
	count := new(int)
	m := &computerManager{owner: cu.SessionOwner{TenantID: 1, UserID: 1}, factory: func(context.Context) (cu.Backend, error) {
		*count++
		return &cu.FakeBackend{CapabilitiesValue: cu.Capabilities{ProtocolVersion: cu.ProtocolVersion, Platform: cu.PlatformMacOS, Backend: cu.BackendNativeHost, CaptureReadiness: cu.ReadinessReady, InputReadiness: cu.ReadinessReady, PermissionState: cu.PermissionApproved, CoordinateSpace: cu.CoordinateSpace{Origin: cu.OriginTopLeft, Unit: cu.CoordinatePixels}}}, nil
	}}
	return m, count
}
func TestComputerStartApprovalAndSnapshotContract(t *testing.T) {
	m, _ := testComputerManager()
	ctx := context.Background()
	pending, err := m.start(ctx, ComputerSessionStartInput{})
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != cu.SessionPendingApproval {
		t.Fatal("implicit approval")
	}
	ready, err := m.start(ctx, ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if ready.ID != pending.ID || ready.State != cu.SessionReady {
		t.Fatal("pending session could not be approved")
	}
	raw, _ := json.Marshal(ready)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if out["session_id"] != ready.ID {
		t.Fatalf("bridge contract mismatch %s", raw)
	}
	if _, exists := out["id"]; exists {
		t.Fatal("legacy wrong ID field retained")
	}
}
func TestComputerControlsReturnStateAndNewSessionReplacesStoppedBackend(t *testing.T) {
	m, count := testComputerManager()
	ctx := context.Background()
	ready, err := m.start(ctx, ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	paused, err := m.control(ctx, ready.ID, cu.ActionPause)
	if err != nil || paused.State != cu.SessionPaused {
		t.Fatalf("pause %v %v", paused, err)
	}
	resumed, err := m.control(ctx, ready.ID, cu.ActionResume)
	if err != nil || resumed.State != cu.SessionNeedsObservation {
		t.Fatalf("resume %v %v", resumed, err)
	}
	stopped, err := m.control(ctx, ready.ID, cu.ActionStop)
	if err != nil || stopped.State != cu.SessionStopped {
		t.Fatalf("stop %v %v", stopped, err)
	}
	if _, err = m.control(ctx, ready.ID, cu.ActionResume); err == nil {
		t.Fatal("resume bypassed stop")
	}
	next, err := m.start(ctx, ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == ready.ID || *count != 2 {
		t.Fatal("stopped helper reused")
	}
	if _, err = m.control(ctx, ready.ID, cu.ActionStop); err == nil {
		t.Fatal("old session accepted")
	}
}

// Any capture/image read through the status endpoint is a regression: model
// screenshots and observation freshness belong exclusively to the controller.
type snapshotOnlyComputerBackend struct {
	cu.FakeBackend
	t *testing.T
}

func (b *snapshotOnlyComputerBackend) Observe(context.Context, cu.ObserveRequest) (cu.Observation, error) {
	b.t.Fatal("snapshot captured the desktop")
	return cu.Observation{}, nil
}
func (b *snapshotOnlyComputerBackend) ObservationImage(context.Context, string) ([]byte, string, error) {
	b.t.Fatal("snapshot accessed model screenshot bytes")
	return nil, "", nil
}

func TestGetComputerSessionReadOnlyModelState(t *testing.T) {
	ctx := context.Background()
	m, _ := testComputerManager()
	backend, err := m.factory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b := &snapshotOnlyComputerBackend{FakeBackend: cu.FakeBackend{
		CapabilitiesValue: backend.(*cu.FakeBackend).CapabilitiesValue,
		ReceiptValue:      cu.ActionReceipt{AfterObservationID: "model-after", After: &cu.MediaRef{ID: "model-after-image"}},
	}, t: t}
	b.CapabilitiesValue.Actions = []cu.ActionKind{cu.ActionWait}
	m.backend = b
	ready, err := m.start(ctx, ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	a := &app{computerManager: m}
	c := m.controller
	s := c.Session()
	o := cu.Observation{ID: "model-observation", SessionID: ready.ID, Width: 100, Height: 100,
		Capabilities: ready.Capabilities, ObservedAt: time.Now(), Screenshot: cu.MediaRef{ID: "model-image"}}
	if err := s.SetObservation(o); err != nil {
		t.Fatal(err)
	}
	before, ok := s.CurrentObservation()
	if !ok {
		t.Fatal("missing model observation")
	}
	for range 3 {
		got, err := a.GetComputerSession(ready.ID)
		if err != nil || got.State != cu.SessionReady || got.Observation == nil || !reflect.DeepEqual(*got.Observation, before) {
			t.Fatalf("snapshot changed model observation: %+v, %v", got, err)
		}
	}
	after, ok := s.CurrentObservation()
	if !ok || !reflect.DeepEqual(before, after) || m.controller != c || !s.Approved() {
		t.Fatal("read changed controller, grant, observation, or freshness")
	}
	action := cu.Action{ID: "model-action", SessionID: ready.ID, ObservationID: o.ID, Kind: cu.ActionWait}
	r, err := c.Execute(ctx, s.Owner(), action)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.GetComputerSession(ready.ID)
	if err != nil || got.State != cu.SessionNeedsObservation || got.LastReceipt == nil || !reflect.DeepEqual(*got.LastReceipt, r) {
		t.Fatalf("model receipt missing: %+v, %v", got, err)
	}
	if _, ok := s.CurrentObservation(); ok {
		t.Fatal("status read restored an observation consumed by the model")
	}
	// Simulate model/runtime cleanup through the shared controller, not UI Stop.
	if err := c.Stop(ctx, s.Owner(), ready.ID); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetComputerSession(ready.ID)
	if err != nil || got.State != cu.SessionStopped || got.LastReceipt == nil || s.Approved() || !b.Stopped {
		t.Fatalf("model Stop not reflected: %+v, %v", got, err)
	}
	if _, err := a.GetComputerSession("unknown-session"); err == nil {
		t.Fatal("accepted a different session ID")
	}
	next, err := m.start(ctx, ComputerSessionStartInput{Approved: true})
	if err != nil || next.ID == ready.ID {
		t.Fatalf("replacement start: %+v, %v", next, err)
	}
	if _, err := a.GetComputerSession(ready.ID); err == nil {
		t.Fatal("accepted replaced session")
	}
}

func TestGetComputerSessionDoesNotInitializeBackend(t *testing.T) {
	m, count := testComputerManager()
	a := &app{computerManager: m}
	if _, err := a.GetComputerSession("missing"); err == nil {
		t.Fatal("missing session accepted")
	}
	if *count != 0 || m.controller != nil || m.backend != nil {
		t.Fatal("status lookup initialized backend or session")
	}
}

type failingCapabilitiesBackend struct {
	*cu.FakeBackend
	err error
}

func (b *failingCapabilitiesBackend) Capabilities(context.Context) (cu.Capabilities, error) {
	return cu.Capabilities{}, b.err
}

func TestComputerManagerRestartsFailedHelperBeforeCreatingSession(t *testing.T) {
	caps := cu.Capabilities{
		ProtocolVersion:  cu.ProtocolVersion,
		Platform:         cu.PlatformMacOS,
		Backend:          cu.BackendNativeHost,
		CaptureReadiness: cu.ReadinessReady,
		InputReadiness:   cu.ReadinessReady,
		PermissionState:  cu.PermissionApproved,
		CoordinateSpace:  cu.CoordinateSpace{Origin: cu.OriginTopLeft, Unit: cu.CoordinatePixels},
	}
	first := &failingCapabilitiesBackend{FakeBackend: &cu.FakeBackend{CapabilitiesValue: caps}, err: context.DeadlineExceeded}
	second := &cu.FakeBackend{CapabilitiesValue: caps}
	created := 0
	m := &computerManager{owner: cu.SessionOwner{TenantID: 1, UserID: 1}, factory: func(context.Context) (cu.Backend, error) {
		created++
		if created == 1 {
			return first, nil
		}
		return second, nil
	}}
	started, err := m.start(context.Background(), ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if started.State != cu.SessionReady || created != 2 || m.backend != second {
		t.Fatalf("state=%q created=%d backend=%T", started.State, created, m.backend)
	}
	if m.controller == nil || m.controller.Session().Capabilities().PermissionState != cu.PermissionApproved {
		t.Fatal("replacement helper did not bind a ready session")
	}
}
