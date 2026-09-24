package computeruse

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSessionAlwaysRequiresApproval(t *testing.T) {
	s, owner, now := newTestSession(t, true)
	if err := s.SetObservation(readyObservation(s.ID(), now)); err == nil {
		t.Fatal("unapproved observation accepted")
	}
	if err := s.Approve(SessionOwner{TenantID: owner.TenantID, UserID: owner.UserID, SessionID: 99}); err == nil {
		t.Fatal("cross-conversation approval")
	}
	if s.Approved() {
		t.Fatal("approval was implicit")
	}
}
func TestStopWinsLateUnknownReceipt(t *testing.T) {
	s, owner, now := newTestSession(t, false)
	_ = s.SetObservation(readyObservation(s.ID(), now))
	a := Action{ID: "a", SessionID: s.ID(), ObservationID: "obs-1", Kind: ActionType, Text: "test"}
	if err := s.BeginAction(a); err != nil {
		t.Fatal(err)
	}
	_ = s.Stop(owner)
	if err := s.RecordReceipt(ActionReceipt{ActionID: a.ID, SessionID: s.ID(), Outcome: OutcomeUnknown}); err != nil {
		t.Fatal(err)
	}
	if s.State() != SessionStopped || s.Approved() {
		t.Fatal("late receipt resurrected session")
	}
	o := readyObservation(s.ID(), now)
	o.ID = "new"
	if s.SetObservation(o) == nil {
		t.Fatal("stopped session accepted observation")
	}
}
func TestUnknownCannotReplayAfterFreshObservation(t *testing.T) {
	s, _, now := newTestSession(t, false)
	_ = s.SetObservation(readyObservation(s.ID(), now))
	a := Action{ID: "a", SessionID: s.ID(), ObservationID: "obs-1", Kind: ActionType, Text: "test"}
	_ = s.BeginAction(a)
	_ = s.RecordReceipt(ActionReceipt{ActionID: a.ID, SessionID: s.ID(), Outcome: OutcomeUnknown})
	if s.SetObservation(readyObservation(s.ID(), now)) == nil {
		t.Fatal("old observation reused")
	}
	o := readyObservation(s.ID(), now)
	o.ID = "fresh"
	if err := s.SetObservation(o); err != nil {
		t.Fatal(err)
	}
	a.ObservationID = o.ID
	if s.BeginAction(a) == nil {
		t.Fatal("unknown action replayed")
	}
	a.ID = "new"
	if err := s.BeginAction(a); err != nil {
		t.Fatal(err)
	}
}
func TestSessionSnapshotDoesNotMutateAuthority(t *testing.T) {
	s, _, now := newTestSession(t, false)
	_ = s.SetObservation(readyObservation(s.ID(), now))
	caps := s.Capabilities()
	caps.Actions[0] = ActionStop
	obs, _ := s.CurrentObservation()
	obs.Capabilities.Actions[0] = ActionStop
	if !s.Capabilities().Supports(ActionClick) {
		t.Fatal("snapshot mutated capabilities")
	}
}
func TestActionBoundsAndUnknownKind(t *testing.T) {
	now := time.Now()
	o := readyObservation("s", now)
	for _, a := range []Action{
		{Kind: ActionKind("script")}, {Kind: ActionWait, DurationMS: -1}, {Kind: ActionWait, DurationMS: MaxActionDurationMS + 1},
		{Kind: ActionType, Text: "x", DisplayID: "other"}, {Kind: ActionHotkey, Keys: []string{""}}, {Kind: ActionScroll, DeltaY: MaxScrollDelta + 1},
	} {
		a.ID = "a"
		a.SessionID = "s"
		a.ObservationID = o.ID
		if a.Validate(now, o) == nil {
			t.Fatalf("invalid action accepted %s", a.Kind)
		}
	}
}
func TestEventCannotSerializeActionText(t *testing.T) {
	a := Action{Kind: ActionType, Text: "private"}
	b, err := json.Marshal(Event{Kind: EventActionRequested, ActionSummary: a.RedactedSummary()})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	_ = json.Unmarshal(b, &fields)
	if _, ok := fields["action"]; ok {
		t.Fatal("event carries raw action")
	}
}

type blockingBackend struct {
	entered chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
}

func (b *blockingBackend) Capabilities(context.Context) (Capabilities, error) {
	return readyCapabilities(), nil
}
func (b *blockingBackend) Observe(context.Context, ObserveRequest) (Observation, error) {
	return Observation{}, errors.New("capture error")
}
func (b *blockingBackend) Execute(ctx context.Context, a Action) (ActionReceipt, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	close(b.entered)
	select {
	case <-ctx.Done():
	case <-b.release:
	}
	return ActionReceipt{}, errors.New("sensitive helper error")
}
func (*blockingBackend) Pause(context.Context) error  { return nil }
func (*blockingBackend) Resume(context.Context) error { return nil }
func (*blockingBackend) Stop(context.Context) error   { return nil }
func (*blockingBackend) Close(context.Context) error  { return nil }
func TestControllerStopCancelsBlockedActionAndRejectsQueuedAction(t *testing.T) {
	s, owner, now := newTestSession(t, false)
	_ = s.SetObservation(readyObservation(s.ID(), now))
	b := &blockingBackend{entered: make(chan struct{}), release: make(chan struct{})}
	c, _ := NewController(s, b)
	a := Action{ID: "a", SessionID: s.ID(), ObservationID: "obs-1", Kind: ActionType, Text: "test"}
	done := make(chan ActionReceipt, 1)
	go func() { r, _ := c.Execute(context.Background(), owner, a); done <- r }()
	<-b.entered
	stopped := make(chan error, 1)
	go func() { stopped <- c.Stop(context.Background(), owner, s.ID()) }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop blocked by Execute")
	}
	r := <-done
	if r.Outcome != OutcomeUnknown || r.ErrorMessage != "" {
		t.Fatalf("unsafe receipt %+v", r)
	}
	a.ID = "b"
	if _, err := c.Execute(context.Background(), owner, a); err == nil {
		t.Fatal("queued input accepted after Stop")
	}
	if b.calls != 1 || s.State() != SessionStopped {
		t.Fatal("Stop lost priority")
	}
}
func TestCaptureFailurePauses(t *testing.T) {
	s, owner, _ := newTestSession(t, false)
	c, _ := NewController(s, &blockingBackend{})
	if _, err := c.Observe(context.Background(), owner, ObserveRequest{SessionID: s.ID()}); err == nil {
		t.Fatal("expected failure")
	}
	if s.State() != SessionPaused {
		t.Fatal("capture failure did not pause")
	}
}
