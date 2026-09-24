package computeruse

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func readyCapabilities() Capabilities {
	return Capabilities{
		ProtocolVersion:  ProtocolVersion,
		Platform:         PlatformMacOS,
		Backend:          BackendNativeHost,
		CaptureReadiness: ReadinessReady,
		InputReadiness:   ReadinessReady,
		FocusState:       FocusFocused,
		PermissionState:  PermissionApproved,
		CoordinateSpace:  CoordinateSpace{Origin: OriginTopLeft, Unit: CoordinatePixels, Width: 800, Height: 600, ScaleFactor: 2},
		Actions:          []ActionKind{ActionClick, ActionType, ActionWait},
		ImageSupported:   true,
		SupportsPause:    true,
		SupportsStop:     true,
	}
}

func readyObservation(sessionID string, now time.Time) Observation {
	return Observation{
		ID: "obs-1", SessionID: sessionID, Width: 800, Height: 600,
		Capabilities: readyCapabilities(), ObservedAt: now, ExpiresAt: now.Add(time.Minute),
	}
}

func newTestSession(t *testing.T, requireApproval bool) (*ComputerSession, SessionOwner, time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	owner := SessionOwner{TenantID: 7, UserID: 11}
	session, err := NewComputerSession(SessionOptions{ID: "computer-1", Owner: owner, Capabilities: readyCapabilities(), Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if !requireApproval {
		if err := session.Approve(owner); err != nil {
			t.Fatal(err)
		}
	}
	return session, owner, now
}

func TestActionRejectsStaleObservation(t *testing.T) {
	session, _, now := newTestSession(t, false)
	if err := session.SetObservation(readyObservation(session.ID(), now)); err != nil {
		t.Fatal(err)
	}
	action := Action{ID: "action-1", SessionID: session.ID(), ObservationID: "old", Kind: ActionClick, Point: &Point{X: 10, Y: 20}}
	if err := session.ValidateAction(action); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("ValidateAction error = %v, want stale observation error", err)
	}
}

func TestUnknownReceiptRequiresFreshObservationAndNeverReplays(t *testing.T) {
	session, _, now := newTestSession(t, false)
	if err := session.SetObservation(readyObservation(session.ID(), now)); err != nil {
		t.Fatal(err)
	}
	action := Action{ID: "action-1", SessionID: session.ID(), ObservationID: "obs-1", Kind: ActionType, Text: "secret"}
	if err := session.ValidateAction(action); err != nil {
		t.Fatal(err)
	}
	if err := session.BeginAction(action); err != nil {
		t.Fatal(err)
	}
	if err := session.RecordReceipt(ActionReceipt{ActionID: action.ID, SessionID: session.ID(), Outcome: OutcomeUnknown}); err != nil {
		t.Fatal(err)
	}
	if session.State() != SessionNeedsObservation {
		t.Fatalf("state = %s, want %s", session.State(), SessionNeedsObservation)
	}
	if err := session.ValidateAction(action); err == nil {
		t.Fatal("unknown outcome allowed input replay")
	}
	fresh := readyObservation(session.ID(), now)
	fresh.ID = "obs-2"
	if err := session.SetObservation(fresh); err != nil {
		t.Fatal(err)
	}
	action.ID = "action-2"
	if action.ObservationID = fresh.ID; session.ValidateAction(action) != nil {
		t.Fatal("fresh observation did not unblock new evaluation")
	}
}

func TestSessionApprovalOwnershipPauseAndStop(t *testing.T) {
	session, owner, _ := newTestSession(t, true)
	if session.State() != SessionPendingApproval {
		t.Fatalf("state = %s, want pending approval", session.State())
	}
	if err := session.Approve(SessionOwner{TenantID: 7, UserID: 99}); err == nil {
		t.Fatal("wrong owner approved session")
	}
	if err := session.Approve(owner); err != nil {
		t.Fatal(err)
	}
	if err := session.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := session.ValidateAction(Action{ID: "a", SessionID: session.ID(), Kind: ActionObserve}); err == nil {
		t.Fatal("paused session accepted action")
	}
	if err := session.Resume(owner); err != nil {
		t.Fatal(err)
	}
	if err := session.Stop(owner); err != nil {
		t.Fatal(err)
	}
	if session.Approved() || session.State() != SessionStopped {
		t.Fatalf("stopped session = state=%s approved=%v", session.State(), session.Approved())
	}
	if err := session.Resume(owner); err == nil {
		t.Fatal("stopped session resumed")
	}
}

func TestSensitiveActionSummaryDoesNotContainText(t *testing.T) {
	action := Action{Kind: ActionType, Text: "password-123"}
	if got := action.RedactedSummary(); strings.Contains(got, action.Text) || !strings.Contains(got, "text_length") {
		t.Fatalf("redacted summary = %q", got)
	}
	encoded, err := json.Marshal(ActionReceipt{RedactedActionSummary: action.RedactedSummary()})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), action.Text) {
		t.Fatalf("receipt leaked sensitive input: %s", encoded)
	}
}

func TestCapabilitiesAreCrossPlatformNeutral(t *testing.T) {
	caps := readyCapabilities()
	encoded, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"CGEvent", "Quartz", "ScreenCaptureKit", "AXUIElement", "Win32", "SendInput"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("capabilities leaked platform implementation detail %q: %s", forbidden, encoded)
		}
	}
}
