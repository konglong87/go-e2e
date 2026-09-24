package main

import (
	"context"
	"encoding/json"
	"testing"

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
