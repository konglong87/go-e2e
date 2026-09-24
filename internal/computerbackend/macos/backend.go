// Package macos provides the Computer Use backend adapter for the macOS native
// helper. No AppKit/CoreGraphics APIs are imported here; they remain isolated in
// the helper process.
package macos

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbackend/native"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const (
	commandReadiness = "readiness"
	commandObserve   = "observe"
	commandExecute   = "execute"
	commandPause     = "pause"
	commandResume    = "resume"
	commandStop      = "stop"
	commandShutdown  = "shutdown"
)

type Config struct {
	HelperPath     string
	HelperArgs     []string
	HelperEnv      []string
	RequestTimeout time.Duration
	MaxFrameBytes  int
	Now            func() time.Time
}

type Backend struct {
	mu             sync.Mutex
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.Reader
	codec          native.Codec
	config         Config
	requestCounter uint64
	closed         bool
	paused         bool
	stopped        bool
	images         map[string]imageData
	capabilities   cu.Capabilities
}

type imageData struct {
	data      []byte
	mediaType string
}

func New(ctx context.Context, config Config) (*Backend, error) {
	if strings.TrimSpace(config.HelperPath) == "" {
		return nil, errors.New("macOS computer helper path is required")
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 10 * time.Second
	}
	if config.MaxFrameBytes <= 0 {
		config.MaxFrameBytes = 8 * 1024 * 1024
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	cmd := exec.CommandContext(ctx, config.HelperPath, config.HelperArgs...)
	if len(config.HelperEnv) > 0 {
		cmd.Env = append([]string(nil), config.HelperEnv...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open helper stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("open helper stdout: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start macOS computer helper: %w", err)
	}
	backend := &Backend{cmd: cmd, stdin: stdin, stdout: stdout, codec: native.NewCodec(uint32(config.MaxFrameBytes)), config: config, images: map[string]imageData{}}
	capabilities, err := backend.Capabilities(ctx)
	if err != nil {
		_ = backend.Close(context.Background())
		return nil, err
	}
	backend.capabilities = capabilities
	return backend, nil
}

func (b *Backend) Capabilities(ctx context.Context) (cu.Capabilities, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	payload, err := b.requestLocked(ctx, commandReadiness, "", "", nil)
	if err != nil {
		return cu.Capabilities{}, err
	}
	caps := cu.Capabilities{ProtocolVersion: cu.ProtocolVersion, Platform: cu.PlatformMacOS, Backend: cu.BackendNativeHost, CoordinateSpace: cu.CoordinateSpace{Origin: cu.OriginTopLeft, Unit: cu.CoordinatePixels}}
	if value, ok := payload["capture_readiness"].(string); ok {
		caps.CaptureReadiness = cu.Readiness(value)
	}
	if value, ok := payload["input_readiness"].(string); ok {
		caps.InputReadiness = cu.Readiness(value)
	}
	if value, ok := payload["permission_state"].(string); ok {
		caps.PermissionState = cu.PermissionState(value)
	}
	if value, ok := payload["focus_state"].(string); ok {
		caps.FocusState = cu.FocusState(value)
	}
	if value, ok := payload["image_supported"].(bool); ok {
		caps.ImageSupported = value
	}
	if value, ok := payload["supports_pause"].(bool); ok {
		caps.SupportsPause = value
	}
	if value, ok := payload["supports_stop"].(bool); ok {
		caps.SupportsStop = value
	}
	if err := caps.Validate(); err != nil {
		return cu.Capabilities{}, err
	}
	b.capabilities = caps
	return caps, nil
}

func (b *Backend) Observe(ctx context.Context, request cu.ObserveRequest) (cu.Observation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	payload, err := b.requestLocked(ctx, commandObserve, request.SessionID, "", map[string]any{"display_id": request.DisplayID, "window_id": request.WindowID})
	if err != nil {
		return cu.Observation{}, err
	}
	data, mediaType, width, height, err := decodeImage(payload)
	if err != nil {
		return cu.Observation{}, err
	}
	id := fmt.Sprintf("observation-%d", b.requestCounter)
	b.images[id] = imageData{data: data, mediaType: mediaType}
	now := b.config.Now()
	return cu.Observation{ID: id, SessionID: request.SessionID, DisplayID: request.DisplayID, WindowID: request.WindowID, Width: width, Height: height, ScaleFactor: 1, Screenshot: cu.NewMediaRef(id, mediaType, data, width, height), Capabilities: b.capabilities, ObservedAt: now, ExpiresAt: now.Add(b.config.RequestTimeout)}, nil
}

func (b *Backend) ObservationImage(_ context.Context, observationID string) ([]byte, string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	image, ok := b.images[observationID]
	if !ok {
		return nil, "", fmt.Errorf("observation image %q not found", observationID)
	}
	return append([]byte(nil), image.data...), image.mediaType, nil
}

func (b *Backend) Execute(ctx context.Context, action cu.Action) (cu.ActionReceipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	started := b.config.Now()
	beforeID := action.ObservationID
	payload := map[string]any{"kind": action.Kind, "observation_id": action.ObservationID, "display_id": action.DisplayID, "window_id": action.WindowID, "text": action.Text, "key": action.Key, "keys": action.Keys, "delta_x": action.DeltaX, "delta_y": action.DeltaY, "duration_ms": action.DurationMS}
	if action.Point != nil {
		payload["x"], payload["y"] = action.Point.X, action.Point.Y
	}
	responsePayload, err := b.requestLocked(ctx, commandExecute, action.SessionID, action.ID, payload)
	if err != nil {
		return cu.ActionReceipt{ActionID: action.ID, SessionID: action.SessionID, Platform: cu.PlatformMacOS, Backend: cu.BackendNativeHost, BeforeObservationID: beforeID, Outcome: cu.OutcomeUnknown, Verification: cu.VerificationUnknown, RedactedActionSummary: action.RedactedSummary(), ErrorCode: "ipc_unknown", ErrorMessage: err.Error(), Duration: b.config.Now().Sub(started), CompletedAt: b.config.Now()}, err
	}
	receipt := cu.ActionReceipt{ActionID: action.ID, SessionID: action.SessionID, Platform: cu.PlatformMacOS, Backend: cu.BackendNativeHost, BeforeObservationID: beforeID, Outcome: cu.OutcomeExecuted, Verification: cu.VerificationNotChecked, RedactedActionSummary: action.RedactedSummary(), Duration: b.config.Now().Sub(started), CompletedAt: b.config.Now()}
	if outcome, ok := responsePayload["_outcome"].(string); ok {
		receipt.Outcome = cu.Outcome(outcome)
	}
	if data, mediaType, width, height, decodeErr := decodeImage(responsePayload); decodeErr == nil {
		afterID := fmt.Sprintf("observation-after-%d", b.requestCounter)
		b.images[afterID] = imageData{data: data, mediaType: mediaType}
		receipt.AfterObservationID = afterID
		receipt.After = mediaRefPtr(cu.NewMediaRef(afterID, mediaType, data, width, height))
	}
	return receipt, nil
}

func (b *Backend) Pause(ctx context.Context) error  { return b.control(ctx, commandPause) }
func (b *Backend) Resume(ctx context.Context) error { return b.control(ctx, commandResume) }
func (b *Backend) Stop(ctx context.Context) error {
	b.mu.Lock()
	b.stopped = true
	b.mu.Unlock()
	return b.control(ctx, commandStop)
}

func (b *Backend) control(ctx context.Context, command string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.requestLocked(ctx, command, "", "", nil)
	if command == commandPause && err == nil {
		b.paused = true
	}
	if command == commandResume && err == nil {
		b.paused = false
	}
	return err
}

func (b *Backend) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	_, _ = b.requestLocked(ctx, commandShutdown, "", "", nil)
	_ = b.stdin.Close()
	return b.cmd.Wait()
}

func (b *Backend) requestLocked(ctx context.Context, command, sessionID, actionID string, payload map[string]any) (map[string]any, error) {
	if b.closed {
		return nil, errors.New("computer helper is closed")
	}
	b.requestCounter++
	requestID := fmt.Sprintf("request-%d", b.requestCounter)
	if sessionID == "" {
		sessionID = "host"
	}
	if actionID == "" {
		actionID = requestID
	}
	deadline := b.config.Now().Add(b.config.RequestTimeout)
	payloadBytes, err := json.Marshal(payloadOrEmpty(payload))
	if err != nil {
		return nil, err
	}
	request := native.Envelope{ProtocolVersion: native.ProtocolVersion, RequestID: requestID, SessionID: sessionID, ActionID: actionID, Deadline: deadline, Command: command, Payload: payloadBytes}
	if err := b.codec.WriteFrame(b.stdin, request); err != nil {
		return nil, fmt.Errorf("write helper request: %w", err)
	}
	readDone := make(chan struct{})
	var response native.Envelope
	var readErr error
	go func() { response, readErr = b.codec.ReadFrame(b.stdout); close(readDone) }()
	requestCtx, cancel := context.WithTimeout(ctx, b.config.RequestTimeout)
	defer cancel()
	select {
	case <-readDone:
	case <-requestCtx.Done():
		_ = b.cmd.Process.Kill()
		return nil, requestCtx.Err()
	}
	if readErr != nil {
		return nil, fmt.Errorf("read helper response: %w", readErr)
	}
	if response.RequestID != requestID {
		return nil, fmt.Errorf("helper response request id mismatch: got %q want %q", response.RequestID, requestID)
	}
	var envelopePayload map[string]any
	if err := json.Unmarshal(response.Payload, &envelopePayload); err != nil {
		return nil, fmt.Errorf("decode helper response payload: %w", err)
	}
	ok, _ := envelopePayload["ok"].(bool)
	outcome, _ := envelopePayload["outcome"].(string)
	if !ok {
		return nil, fmt.Errorf("helper %v: %v", envelopePayload["error_code"], envelopePayload["error_message"])
	}
	result, _ := envelopePayload["result"].(map[string]any)
	if result == nil {
		result = map[string]any{}
	}
	if outcome != "" {
		result["_outcome"] = outcome
	}
	return result, nil
}

func payloadOrEmpty(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}

func decodeImage(payload map[string]any) ([]byte, string, int, int, error) {
	encoded, ok := payload["data"].(string)
	if !ok || encoded == "" {
		return nil, "", 0, 0, errors.New("helper response did not include screenshot data")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("decode screenshot: %w", err)
	}
	mediaType, _ := payload["media_type"].(string)
	width, _ := payload["width"].(float64)
	height, _ := payload["height"].(float64)
	if mediaType == "" {
		mediaType = "image/png"
	}
	return data, mediaType, int(width), int(height), nil
}

func mediaRefPtr(value cu.MediaRef) *cu.MediaRef { return &value }

func imageHash(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
