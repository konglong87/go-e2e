package macos

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbackend/native"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

func testPNG() []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	return b.Bytes()
}
func imagePayload() map[string]any {
	return map[string]any{"data": base64.StdEncoding.EncodeToString(testPNG()), "media_type": pngMediaType, "width": float64(2), "height": float64(2), "display_id": "1", "scale_factor": float64(2)}
}

// No system APIs: every subprocess below is this Go test binary, not the
// production native helper. Requests/actions never operate the user desktop.
func TestHelperProcess(t *testing.T) {
	args := os.Args
	idx := -1
	for i, a := range args {
		if a == "computer-test-helper" {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	defer syscall.Exit(0) // bypass race runtime one-second exit delay in fixture child
	mode := args[idx+1]
	marker := args[idx+2]
	codec := native.NewCodec(maxFrameBytes)
	for {
		req, err := codec.ReadFrame(os.Stdin)
		if err != nil {
			return
		}
		result := map[string]any{}
		outcome := cu.OutcomeExecuted
		switch req.Command {
		case commandRequestPermissions:
			if marker != "" {
				_ = os.WriteFile(marker, append(readMarker(t, marker), []byte("permissions\n")...), 0600)
			}
			fallthrough
		case commandReadiness:
			result = map[string]any{"capture_readiness": "ready", "input_readiness": "ready", "permission_state": "approved", "focus_state": "focused", "image_supported": true, "supports_pause": true, "supports_stop": true, "coordinate_space": map[string]any{"display_id": "1", "width": 2, "height": 2, "scale_factor": 2}}
		case commandObserve:
			result = imagePayload()
			result["coordinate_space"] = captureGeometryFixture()
			var observeRequest struct {
				WindowID string `json:"window_id"`
			}
			if err := json.Unmarshal(req.Payload, &observeRequest); err != nil {
				t.Fatal(err)
			}
			if mode == "target-window" && observeRequest.WindowID != "" {
				result["coordinate_space"].(map[string]any)["bounds"] = map[string]any{"x": 10, "y": 20, "width": 1, "height": 1}
				result["window_id"] = "window-1"
				result["target_window"] = map[string]any{"id": "window-1", "title": "Fixture", "owner_pid": 42, "bundle_id": "fixture.app", "frame": map[string]any{"x": 10, "y": 20, "width": 1, "height": 1}, "is_visible": true}
			}
			mutateCaptureFixture(mode, result)
			result["observation_expires_at"] = time.Now().Add(cu.DefaultObservationTTL).Format(time.RFC3339Nano)
			switch mode {
			case "missing-expiry":
				delete(result, "observation_expires_at")
			case "invalid-expiry":
				result["observation_expires_at"] = "not-a-time"
			case "past-expiry":
				result["observation_expires_at"] = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
			case "long-expiry":
				result["observation_expires_at"] = time.Now().Add(time.Hour).Format(time.RFC3339Nano)
			}
		case commandExecute:
			if marker != "" {
				_ = os.WriteFile(marker, []byte("execute"), 0600)
			}
			switch mode {
			case "hang":
				continue
			case "exit":
				return
			case "reject":
				outcome = cu.OutcomeRejected
			case "unknown", "focus-failed":
				outcome = cu.OutcomeUnknown
			case "failed":
				outcome = cu.OutcomeFailed
			case "invalid-image":
				result = imagePayload()
				result["data"] = base64.StdEncoding.EncodeToString([]byte("png"))
			default:
				result = imagePayload()
			}
			switch mode {
			case "session":
				req.SessionID = "other"
			case "action":
				req.ActionID = "other"
			case "request":
				req.RequestID = "other"
			case "command":
				req.Command = "other"
			case "deadline":
				req.Deadline = req.Deadline.Add(time.Second)
			}
		case commandShutdown:
			if mode == "shutdown-marker" {
				_ = os.WriteFile(marker, []byte("shutdown"), 0600)
			}
			if mode == "ignore-shutdown" {
				time.Sleep(time.Hour)
				return
			}
		case "environment":
			result["env"] = os.Environ()
		case "stderr":
			_, _ = os.Stderr.WriteString(strings.Repeat("SECRET-CREDENTIAL", 100000))
		case "block":
			continue
		}
		errorCode := ""
		if mode == "focus-failed" && req.Command == commandExecute {
			errorCode = helperFocusChangedCode
		}
		payload, _ := json.Marshal(map[string]any{"ok": outcome == cu.OutcomeExecuted, "outcome": outcome, "result": result, "error_code": errorCode, "error_message": "SECRET-CREDENTIAL"})
		req.Payload = payload
		_ = codec.WriteFrame(os.Stdout, req)
		if req.Command == commandShutdown {
			return
		}
	}
}

func readMarker(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	return data
}
func helperConfig(t *testing.T, mode string, timeout time.Duration, marker string) Config {
	t.Helper()
	return Config{HelperPath: os.Args[0], HelperArgs: []string{"-test.run=^TestHelperProcess$", "--", "computer-test-helper", mode, marker}, RequestTimeout: timeout}
}
func newTestBackend(t *testing.T, mode string, timeout time.Duration, marker string) *Backend {
	t.Helper()
	b, err := New(context.Background(), helperConfig(t, mode, timeout, marker))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close(context.Background()) })
	return b
}
func observeTest(t *testing.T, b *Backend) cu.Observation {
	t.Helper()
	obs, err := b.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1"})
	if err != nil {
		t.Fatal(err)
	}
	return obs
}
func waitAction(obs cu.Observation) cu.Action {
	return cu.Action{ID: "action-1", SessionID: obs.SessionID, ObservationID: obs.ID, Kind: cu.ActionWait, DurationMS: 1}
}
func awaitMarker(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("helper did not receive execute")
}

func TestNewCallsHostPermissionRequest(t *testing.T) {
	requested := false
	cfg := helperConfig(t, "", time.Second, "")
	cfg.RequestHostPermissions = func() { requested = true }
	b, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close(context.Background()) }()
	if !requested {
		t.Fatal("host permission request was not invoked")
	}
}

func TestCapabilitiesRefreshesPermissionProbe(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "permissions")
	b := newTestBackend(t, "", time.Second, marker)
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Capabilities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Capabilities(context.Background()); err != nil {
		t.Fatal(err)
	}
	data := readMarker(t, marker)
	if got := strings.Count(string(data), "permissions\n"); got != 2 {
		t.Fatalf("permission probe count = %d, want 2; marker=%q", got, data)
	}
}

func TestBackendObserveExecuteAndStop(t *testing.T) {
	b := newTestBackend(t, "", time.Second, "")
	caps, err := b.Capabilities(context.Background())
	if err != nil || !caps.Ready() || !caps.Supports(cu.ActionClick) || caps.CoordinateSpace.ScaleFactor != 2 {
		t.Fatalf("caps=%+v err=%v", caps, err)
	}
	obs := observeTest(t, b)
	if obs.ScaleFactor != 2 || obs.DisplayID != "1" || obs.Capabilities.CoordinateSpace.Width != 2 {
		t.Fatalf("bad observation: %+v", obs)
	}
	data, media, err := b.ObservationImage(context.Background(), obs.ID)
	if err != nil || !bytes.Equal(data, testPNG()) || media != pngMediaType {
		t.Fatalf("image error %v", err)
	}
	data[0] = 0
	again, _, _ := b.ObservationImage(context.Background(), obs.ID)
	if again[0] == 0 {
		t.Fatal("mutable image escaped")
	}
	receipt, err := b.Execute(context.Background(), waitAction(obs))
	if err != nil || receipt.Outcome != cu.OutcomeExecuted || receipt.After == nil {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if _, err = b.Execute(context.Background(), waitAction(obs)); err == nil {
		t.Fatal("replayed action")
	}
	_ = observeTest(t, b)
	if _, _, err = b.ObservationImage(context.Background(), obs.ID); err == nil {
		t.Fatal("old PNG retained")
	}
	if err = b.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1"}); err == nil {
		t.Fatal("observe after stop")
	}
	if err = b.Resume(context.Background()); err == nil {
		t.Fatal("resume after stop")
	}
}

func TestObserveBindsTargetWindowMetadata(t *testing.T) {
	b := newTestBackend(t, "target-window", time.Second, "")
	obs, err := b.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1", WindowID: "window-1"})
	if err != nil {
		t.Fatal(err)
	}
	if obs.WindowID != "window-1" || obs.Capabilities.TargetWindow.ID != "window-1" {
		t.Fatalf("target window not bound: observation=%+v target=%+v", obs, obs.Capabilities.TargetWindow)
	}
	if obs.Capabilities.TargetWindow.BundleID != "fixture.app" || obs.Capabilities.TargetWindow.Frame == nil {
		t.Fatalf("target metadata incomplete: %+v", obs.Capabilities.TargetWindow)
	}
	if *obs.Capabilities.TargetWindow.Frame != *obs.Capabilities.CoordinateSpace.Bounds {
		t.Fatal("window frame differs from capture geometry")
	}
	obs.Capabilities.TargetWindow.Frame.X = 999
	b.mu.Lock()
	isolated := b.observation.Capabilities.TargetWindow.Frame.X == 10 && b.capabilities.TargetWindow.Frame.X == 10
	b.mu.Unlock()
	if !isolated {
		t.Fatal("window target aliases backend snapshot")
	}
	display := observeTest(t, b)
	if display.WindowID != "" || display.Capabilities.TargetWindow != (cu.WindowRef{}) {
		t.Fatal("window target survived transition to display capture")
	}
}

func TestBackendFailureClassification(t *testing.T) {
	for _, mode := range []string{"hang", "exit", "session", "action", "request", "command", "deadline", "unknown", "failed", "invalid-image", "reject"} {
		t.Run(mode, func(t *testing.T) {
			b := newTestBackend(t, mode, time.Second, "")
			obs := observeTest(t, b)
			ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
			defer cancel()
			receipt, err := b.Execute(ctx, waitAction(obs))
			want := cu.OutcomeUnknown
			if mode == "reject" {
				want = cu.OutcomeRejected
			}
			if err == nil || receipt.Outcome != want {
				t.Fatalf("outcome=%s want=%s err=%v", receipt.Outcome, want, err)
			}
			if strings.Contains(err.Error(), "SECRET") || strings.Contains(receipt.ErrorMessage, "SECRET") {
				t.Fatal("sensitive helper error escaped")
			}
			if mode != "reject" {
				if _, err = b.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1"}); err == nil {
					t.Fatal("uncertain helper reused")
				}
			}
		})
	}
}

func TestBackendControlsDoNotWaitForExecute(t *testing.T) {
	for _, command := range []string{commandStop, commandPause, commandShutdown} {
		t.Run(command, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "started")
			b := newTestBackend(t, "hang", 5*time.Second, marker)
			obs := observeTest(t, b)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})
			go func() { _, _ = b.Execute(ctx, waitAction(obs)); close(done) }()
			awaitMarker(t, marker)
			start := time.Now()
			var err error
			switch command {
			case commandStop:
				err = b.Stop(context.Background())
			case commandPause:
				err = b.Pause(context.Background())
			case commandShutdown:
				err = b.Close(context.Background())
			}
			if err != nil {
				t.Fatal(err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("control blocked by action")
			}
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("inflight call leaked")
			}
		})
	}
}

func TestBackendCloseSendsShutdownAndIsBounded(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "closed")
	b := newTestBackend(t, "shutdown-marker", time.Second, marker)
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "shutdown" {
		t.Fatalf("shutdown missing: %v", err)
	}
	if err = b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	b = newTestBackend(t, "ignore-shutdown", time.Second, "")
	start := time.Now()
	_ = b.Close(context.Background())
	if time.Since(start) > 2*time.Second {
		t.Fatal("close unbounded")
	}
}

func TestBackendCallerCancellationAndLocalRejections(t *testing.T) {
	b := newTestBackend(t, "", time.Second, "")
	obs := observeTest(t, b)
	action := waitAction(obs)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	receipt, err := b.Execute(ctx, action)
	if !errors.Is(err, context.Canceled) || receipt.Outcome != cu.OutcomeRejected {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	action.SessionID = "other"
	receipt, err = b.Execute(context.Background(), action)
	if err == nil || receipt.Outcome != cu.OutcomeRejected {
		t.Fatal("cross-session input accepted")
	}
	action = waitAction(obs)
	action.DisplayID = "other"
	if _, err = b.Execute(context.Background(), action); err == nil {
		t.Fatal("wrong display")
	}
	action = waitAction(obs)
	action.ObservationID = "stale"
	if _, err = b.Execute(context.Background(), action); err == nil {
		t.Fatal("stale observation")
	}
	if err = b.Pause(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Execute(context.Background(), waitAction(obs)); err == nil {
		t.Fatal("paused input")
	}
	if err = b.Resume(context.Background()); err != nil {
		t.Fatal(err)
	}
	_ = observeTest(t, b)
}

func TestBackendEnvironmentAndStderr(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "SECRET-CREDENTIAL")
	t.Setenv("CUSTOM_CREDENTIAL", "SECRET-CREDENTIAL")
	t.Setenv("SSH_AUTH_SOCK", "SECRET-SOCKET")
	t.Setenv("DYLD_INSERT_LIBRARIES", "")
	cfg := helperConfig(t, "", time.Second, "")
	cfg.HelperEnv = []string{"ANTHROPIC_API_KEY=SECRET-CREDENTIAL", "DYLD_LIBRARY_PATH=/unsafe", "HOME=" + os.Getenv("HOME")}
	b, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close(context.Background())
	response, err := b.request(context.Background(), "environment", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(response.Result)
	for _, bad := range []string{"SECRET", "API_KEY", "DYLD", "SSH_AUTH", "CUSTOM_CREDENTIAL"} {
		if bytes.Contains(data, []byte(bad)) {
			t.Fatalf("environment leaked %s", bad)
		}
	}
	if _, err = b.request(context.Background(), "stderr", "", "", nil); err != nil {
		t.Fatal("stderr backpressure", err)
	}
}

func TestBackendRejectsUnsafeLimits(t *testing.T) {
	for _, c := range []Config{{HelperPath: "none", MaxFrameBytes: 1 << 40}, {HelperPath: "none", RequestTimeout: time.Hour}} {
		if _, err := New(context.Background(), c); err == nil {
			t.Fatal("unsafe limit accepted")
		}
	}
}

func TestObservationLifetimeIsNotCaptureRPCTimeout(t *testing.T) {
	b := newTestBackend(t, "", time.Second, "")
	obs := observeTest(t, b)
	lifetime := obs.ExpiresAt.Sub(obs.ObservedAt)
	if lifetime < cu.DefaultObservationTTL-time.Second || lifetime > cu.DefaultObservationTTL {
		t.Fatalf("unexpected snapshot lifetime %s", lifetime)
	}
	// Advance only the host's domain clock; no sleeps or increased RPC deadlines.
	// The previous timeout-bound observation would reject this before dispatch.
	b.config.Now = func() time.Time { return obs.ObservedAt.Add(10 * time.Second) }
	receipt, err := b.Execute(context.Background(), waitAction(obs))
	if err != nil || receipt.Outcome != cu.OutcomeExecuted {
		t.Fatalf("fresh observation after capture timeout: %+v %v", receipt, err)
	}
}
func TestObservationExpirationMetadataFailsClosed(t *testing.T) {
	for _, mode := range []string{"missing-expiry", "invalid-expiry", "past-expiry", "long-expiry"} {
		t.Run(mode, func(t *testing.T) {
			b := newTestBackend(t, mode, time.Second, "")
			if _, err := b.Observe(context.Background(), cu.ObserveRequest{SessionID: "session-1"}); err == nil {
				t.Fatal("unbounded/missing expiration accepted")
			}
		})
	}
}

func TestCooperativeInterruptionRequiresLocalControlAndNativeAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name           string
		paused, closed bool
		epoch          uint64
		outcome        cu.Outcome
		code           string
		want           bool
	}{
		{"local pause", true, false, 1, cu.OutcomeUnknown, helperInactiveCode, true},
		{"same generation", true, false, 0, cu.OutcomeUnknown, helperInactiveCode, false},
		{"no local pause", false, false, 1, cu.OutcomeUnknown, helperInactiveCode, false},
		{"closed", true, true, 1, cu.OutcomeUnknown, helperInactiveCode, false},
		{"focus failure", true, false, 1, cu.OutcomeUnknown, "focus_changed", false},
		{"missing response", true, false, 1, "", "", false},
		{"wrong outcome", true, false, 1, cu.OutcomeExecuted, helperInactiveCode, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &Backend{paused: tc.paused, closed: tc.closed, epoch: tc.epoch}
			ok := false
			if got := b.cooperativeInterruption(0, helperResponse{OK: &ok, Outcome: tc.outcome, ErrorCode: tc.code}); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCooperativeInterruptionRejectsMalformedNativeResponse(t *testing.T) {
	b := &Backend{paused: true, epoch: 1}
	ok := true
	for _, flag := range []*bool{nil, &ok} {
		if b.cooperativeInterruption(0, helperResponse{OK: flag, Outcome: cu.OutcomeUnknown, ErrorCode: helperInactiveCode}) {
			t.Fatal("malformed response retained helper")
		}
	}
}

func TestAcknowledgedFocusFailureRetainsOnlyStopTransport(t *testing.T) {
	b := newTestBackend(t, "focus-failed", time.Second, "")
	obs := observeTest(t, b)
	receipt, err := b.Execute(context.Background(), waitAction(obs))
	if err == nil || receipt.Outcome != cu.OutcomeUnknown {
		t.Fatalf("lost uncertainty: %+v %v", receipt, err)
	}
	if err := b.Resume(context.Background()); err == nil {
		t.Fatal("resumed after focus failure")
	}
	if _, err := b.Observe(context.Background(), cu.ObserveRequest{SessionID: obs.SessionID}); err == nil {
		t.Fatal("observed after focus failure")
	}
	if err := b.Stop(context.Background()); err != nil {
		t.Fatalf("lost cleanup-only transport: %v", err)
	}
}
