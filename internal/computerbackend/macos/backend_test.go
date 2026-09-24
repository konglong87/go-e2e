package macos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbackend/native"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_COMPUTER_HELPER") != "1" {
		return
	}
	defer os.Exit(0)
	onePixel := base64.StdEncoding.EncodeToString([]byte("png"))
	codec := native.NewCodec(8 * 1024 * 1024)
	for {
		request, err := codec.ReadFrame(os.Stdin)
		if err != nil {
			return
		}
		if request.Command == commandShutdown {
			writeTestResponse(codec, request, true, cu.OutcomeExecuted, nil)
			return
		}
		if request.Command == "block" {
			time.Sleep(time.Second)
			continue
		}
		result := map[string]any{}
		switch request.Command {
		case commandReadiness:
			result = map[string]any{"capture_readiness": "ready", "input_readiness": "ready", "permission_state": "approved", "focus_state": "focused", "image_supported": true, "supports_pause": true, "supports_stop": true}
		case commandObserve, commandExecute:
			result = map[string]any{"media_type": "image/png", "data": onePixel, "width": 1, "height": 1}
		}
		writeTestResponse(codec, request, true, cu.OutcomeExecuted, result)
	}
}

func writeTestResponse(codec native.Codec, request native.Envelope, ok bool, outcome cu.Outcome, result map[string]any) {
	payload, _ := json.Marshal(map[string]any{"ok": ok, "outcome": outcome, "result": result})
	_ = codec.WriteFrame(os.Stdout, native.Envelope{ProtocolVersion: native.ProtocolVersion, RequestID: request.RequestID, SessionID: request.SessionID, ActionID: request.ActionID, Deadline: time.Now().Add(time.Second), Command: request.Command, Payload: payload})
}

func helperConfig(t *testing.T, timeout time.Duration) Config {
	t.Helper()
	return Config{HelperPath: os.Args[0], HelperArgs: []string{"-test.run=TestHelperProcess"}, HelperEnv: append(os.Environ(), "GO_WANT_COMPUTER_HELPER=1"), RequestTimeout: timeout}
}

func TestBackendObserveExecuteAndStop(t *testing.T) {
	backend, err := New(context.Background(), helperConfig(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close(context.Background())
	caps, err := backend.Capabilities(context.Background())
	if err != nil || !caps.Ready() {
		t.Fatalf("caps=%+v err=%v", caps, err)
	}
	observation, err := backend.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1"})
	if err != nil || observation.Screenshot.SHA256 == "" {
		t.Fatalf("observation=%+v err=%v", observation, err)
	}
	image, mediaType, err := backend.ObservationImage(context.Background(), observation.ID)
	if err != nil || string(image) != "png" || mediaType != "image/png" {
		t.Fatalf("image=%q type=%q err=%v", image, mediaType, err)
	}
	receipt, err := backend.Execute(context.Background(), cu.Action{ID: "action-1", SessionID: "session-1", ObservationID: observation.ID, Kind: cu.ActionWait, DurationMS: 1})
	if err != nil || receipt.Outcome != cu.OutcomeExecuted || receipt.After == nil {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if err := backend.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBackendTimeoutReturnsUnknownOutcome(t *testing.T) {
	backend, err := New(context.Background(), helperConfig(t, 10*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close(context.Background())
	_, err = backend.requestLocked(context.Background(), "block", "session-1", "action-1", nil)
	if err == nil || (!errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline")) {
		t.Fatalf("timeout error = %v", err)
	}
}
