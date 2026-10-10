package computeruse

import (
	"context"
	"errors"
	"testing"
	"time"
)

type turnImageBackend struct {
	*FakeBackend
	imageData   []byte
	mediaType   string
	imageErr    error
	imageCalls  int
	lastImageID string
}

func (b *turnImageBackend) ObservationImage(_ context.Context, observationID string) ([]byte, string, error) {
	b.imageCalls++
	b.lastImageID = observationID
	if b.imageErr != nil {
		return nil, "", b.imageErr
	}
	return b.imageData, b.mediaType, nil
}

type postObserveFailureBackend struct {
	*FakeBackend
	observeCalls int
	observeErr   error
}

func (b *postObserveFailureBackend) Observe(context.Context, ObserveRequest) (Observation, error) {
	b.observeCalls++
	return Observation{}, b.observeErr
}

func prepareTurnController(t *testing.T, backend Backend) (*Controller, *ComputerSession, SessionOwner, Observation) {
	t.Helper()
	session, owner, now := newTestSession(t, false)
	initial := readyObservation(session.ID(), now)
	initial.ID = "obs-before"
	if err := session.SetObservation(initial); err != nil {
		t.Fatal(err)
	}
	controller, err := NewController(session, backend)
	if err != nil {
		t.Fatal(err)
	}
	return controller, session, owner, initial
}

func TestControllerExecuteTurnReturnsReceiptObservationAndScreenshotFromOneTurn(t *testing.T) {
	backend := &turnImageBackend{
		FakeBackend: &FakeBackend{
			ObservationValue: readyObservation("", initialTestTime()),
			ReceiptValue:     ActionReceipt{Outcome: OutcomeExecuted, DispatchState: DispatchComplete},
		},
		imageData: []byte("png-bytes"),
		mediaType: "image/png",
	}
	backend.ObservationValue.ID = "obs-after"
	controller, session, owner, initial := prepareTurnController(t, backend)
	action := Action{ID: "action-turn", SessionID: session.ID(), ObservationID: initial.ID, Kind: ActionWait}

	result, err := controller.ExecuteTurn(context.Background(), owner, action)
	if err != nil {
		t.Fatalf("ExecuteTurn() error = %v", err)
	}
	if result.Receipt.ActionID != action.ID || result.Receipt.SessionID != session.ID() {
		t.Fatalf("receipt = %+v, want action/session binding", result.Receipt)
	}
	if result.Outcome != OutcomeExecuted || result.DispatchState != DispatchComplete {
		t.Fatalf("turn state = outcome=%s dispatch=%s", result.Outcome, result.DispatchState)
	}
	if result.ObservationState != ObservationReady || result.Observation == nil || result.Observation.ID != "obs-after" {
		t.Fatalf("observation = state=%s value=%+v", result.ObservationState, result.Observation)
	}
	if result.ScreenshotState != ScreenshotReady || result.Screenshot == nil {
		t.Fatalf("screenshot = state=%s value=%+v", result.ScreenshotState, result.Screenshot)
	}
	if result.Screenshot.MediaType != "image/png" || result.Screenshot.SHA256 == "" {
		t.Fatalf("screenshot metadata = %+v", result.Screenshot)
	}
	if backend.imageCalls != 1 || backend.lastImageID != "obs-after" {
		t.Fatalf("image calls = %d id=%q, want one read for next observation", backend.imageCalls, backend.lastImageID)
	}
	stored, ok := session.CurrentTurnResult()
	if !ok || stored.TurnID != result.TurnID {
		t.Fatalf("stored turn = %+v, ok=%v", stored, ok)
	}
}

func TestControllerExecuteTurnScreenshotFailureKeepsExecutedReceipt(t *testing.T) {
	backend := &turnImageBackend{
		FakeBackend: &FakeBackend{
			ObservationValue: readyObservation("", initialTestTime()),
			ReceiptValue:     ActionReceipt{Outcome: OutcomeExecuted, DispatchState: DispatchComplete},
		},
		imageErr: errors.New("decode failed"),
	}
	backend.ObservationValue.ID = "obs-after"
	controller, session, owner, initial := prepareTurnController(t, backend)
	action := Action{ID: "action-image-fail", SessionID: session.ID(), ObservationID: initial.ID, Kind: ActionWait}

	result, err := controller.ExecuteTurn(context.Background(), owner, action)
	if err == nil {
		t.Fatal("ExecuteTurn() error = nil, want screenshot failure")
	}
	if result.Outcome != OutcomeExecuted || result.DispatchState != DispatchComplete {
		t.Fatalf("receipt was downgraded after screenshot failure: %+v", result)
	}
	if result.Receipt.ActionID == "" || result.Receipt.SessionID == "" || result.Receipt.Outcome != OutcomeExecuted {
		t.Fatalf("screenshot failure lost receipt: %+v", result.Receipt)
	}
	if result.ScreenshotState != ScreenshotUnavailable || result.RetryPolicy != RetryObserveOnly {
		t.Fatalf("screenshot failure policy = state=%s retry=%s", result.ScreenshotState, result.RetryPolicy)
	}
	if result.ObservationState != ObservationReady || result.Observation == nil {
		t.Fatalf("observation authority was lost with screenshot: state=%s value=%+v", result.ObservationState, result.Observation)
	}
}

func TestControllerExecuteTurnPostObserveFailureDoesNotRejectExecutedAction(t *testing.T) {
	backend := &postObserveFailureBackend{
		FakeBackend: &FakeBackend{
			ObservationValue: readyObservation("", initialTestTime()),
			ReceiptValue:     ActionReceipt{Outcome: OutcomeExecuted, DispatchState: DispatchComplete},
		},
		observeErr: errors.New("capture unavailable"),
	}
	backend.ObservationValue.ID = "obs-after"
	controller, session, owner, initial := prepareTurnController(t, backend)
	action := Action{ID: "action-observe-fail", SessionID: session.ID(), ObservationID: initial.ID, Kind: ActionWait}

	result, err := controller.ExecuteTurn(context.Background(), owner, action)
	if err == nil {
		t.Fatal("ExecuteTurn() error = nil, want observation failure")
	}
	if result.Outcome != OutcomeExecuted || result.Receipt.Outcome != OutcomeExecuted {
		t.Fatalf("post-action observe failure rejected executed action: %+v", result)
	}
	if result.ObservationState != ObservationUnavailable || result.RetryPolicy != RetryObserveOnly {
		t.Fatalf("observation failure policy = state=%s retry=%s", result.ObservationState, result.RetryPolicy)
	}
	if result.ScreenshotState != ScreenshotUnavailable {
		t.Fatalf("observation failure screenshot state = %s", result.ScreenshotState)
	}
	if session.State() == SessionFailed {
		t.Fatal("post-action observation failure permanently failed the session")
	}
}

func TestControllerExecuteTurnUnknownDispatchNeverReplays(t *testing.T) {
	backend := &FakeBackend{
		ObservationValue: readyObservation("", initialTestTime()),
		ReceiptValue:     ActionReceipt{Outcome: OutcomeUnknown, DispatchState: DispatchUnknown},
		ExecuteError:     errors.New("bridge response lost"),
	}
	backend.ObservationValue.ID = "obs-after"
	controller, session, owner, initial := prepareTurnController(t, backend)
	action := Action{ID: "action-unknown", SessionID: session.ID(), ObservationID: initial.ID, Kind: ActionWait}

	result, err := controller.ExecuteTurn(context.Background(), owner, action)
	if err == nil {
		t.Fatal("ExecuteTurn() error = nil, want unknown dispatch error")
	}
	if result.Outcome != OutcomeUnknown || result.DispatchState != DispatchUnknown || result.RetryPolicy != RetryNever {
		t.Fatalf("unknown result = %+v", result)
	}
	if result.Receipt.ActionID != action.ID || result.Receipt.SessionID != session.ID() || result.Receipt.Outcome != OutcomeUnknown {
		t.Fatalf("unknown result lost non-zero receipt: %+v", result.Receipt)
	}
	if len(backend.Executed) != 1 {
		t.Fatalf("backend execution count = %d, want one dispatch", len(backend.Executed))
	}
}

func initialTestTime() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) }

type recordingTurnPublisher struct {
	results chan ComputerTurnResult
}

func (p *recordingTurnPublisher) PublishTurn(result ComputerTurnResult) {
	p.results <- result
}

func TestControllerExecuteTurnPublishesCommittedResultWithoutReconstruction(t *testing.T) {
	backend := &turnImageBackend{
		FakeBackend: &FakeBackend{
			ObservationValue: readyObservation("", initialTestTime()),
			ReceiptValue:     ActionReceipt{Outcome: OutcomeExecuted, DispatchState: DispatchComplete},
		},
		imageData: []byte("png-bytes"),
		mediaType: "image/png",
	}
	backend.ObservationValue.ID = "obs-after"
	controller, session, owner, initial := prepareTurnController(t, backend)
	publisher := &recordingTurnPublisher{results: make(chan ComputerTurnResult, 1)}
	controller.SetTurnPublisher(publisher)
	action := Action{ID: "action-published", SessionID: session.ID(), ObservationID: initial.ID, Kind: ActionWait}

	result, err := controller.ExecuteTurn(context.Background(), owner, action)
	if err != nil {
		t.Fatalf("ExecuteTurn() error = %v", err)
	}
	published := <-publisher.results
	if published.TurnID != result.TurnID || published.ActionID != result.ActionID || published.Receipt.ActionID != result.Receipt.ActionID {
		t.Fatalf("published = %+v, returned = %+v", published, result)
	}
	stored, ok := session.CurrentTurnResult()
	if !ok || stored.TurnID != published.TurnID {
		t.Fatalf("published result was not committed before event: stored=%+v ok=%v", stored, ok)
	}
}
