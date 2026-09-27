// Package macos isolates the native helper from the Computer Use contract.
package macos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbackend/native"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const (
	commandRequestPermissions = "request_permissions"
	commandReadiness          = "readiness"
	commandObserve            = "observe"
	commandExecute            = "execute"
	commandPause              = "pause"
	commandResume             = "resume"
	commandStop               = "stop"
	commandShutdown           = "shutdown"
	defaultTimeout            = 10 * time.Second
	maxTimeout                = 60 * time.Second
	maxFrameBytes             = 8 * 1024 * 1024
	maxActions                = 4096
)

type Config struct {
	HelperPath             string
	HelperArgs             []string
	HelperEnv              []string
	RequestTimeout         time.Duration
	RequestHostPermissions func()
	MaxFrameBytes          int
	Now                    func() time.Time
}

type Backend struct {
	mu                      sync.Mutex // state only, never held across IPC
	operation               chan struct{}
	process                 *native.Process
	config                  Config
	requestCounter          atomic.Uint64
	closed, paused, stopped bool
	epoch                   uint64
	images                  map[string]imageData
	observation             cu.Observation
	actions                 map[string]struct{}
	capabilities            cu.Capabilities
}
type imageData struct {
	data      []byte
	mediaType string
}
type helperResponse struct {
	OK        *bool          `json:"ok"`
	Outcome   cu.Outcome     `json:"outcome"`
	Result    map[string]any `json:"result"`
	ErrorCode string         `json:"error_code"`
	// Intentionally do not decode or surface arbitrary helper error_message.
}
type rejection struct{ code string }

func (e *rejection) Error() string { return "computer helper rejected request: " + e.code }

// Native input needs no model credentials, agent sockets or loader overrides.
// Apply the same allowlist to explicit env overrides, not just inherited env.
func helperEnvironment(extra []string) []string {
	allowed := map[string]bool{"HOME": true, "USER": true, "LOGNAME": true, "TMPDIR": true, "LANG": true, "LC_CTYPE": true, "__CF_USER_TEXT_ENCODING": true}
	values := map[string]string{"PATH": "/usr/bin:/bin:/usr/sbin:/sbin"}
	for _, entry := range append(os.Environ(), extra...) {
		k, v, ok := strings.Cut(entry, "=")
		if ok && allowed[k] {
			values[k] = v
		}
	}
	env := make([]string, 0, len(values))
	for k, v := range values {
		env = append(env, k+"="+v)
	}
	return env
}

func New(ctx context.Context, c Config) (*Backend, error) {
	if strings.TrimSpace(c.HelperPath) == "" {
		return nil, errors.New("macOS computer helper path is required")
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = defaultTimeout
	}
	if c.RequestTimeout > maxTimeout {
		return nil, errors.New("helper request timeout exceeds maximum")
	}
	if c.MaxFrameBytes <= 0 {
		c.MaxFrameBytes = maxFrameBytes
	}
	if c.MaxFrameBytes > maxFrameBytes {
		return nil, errors.New("helper frame limit exceeds maximum")
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	// Request from the signed desktop host as well as the nested helper. macOS
	// attributes TCC prompts to the process making the request; keeping this
	// in the host process makes the visible go-e2e entry the source of truth.
	if c.RequestHostPermissions != nil {
		c.RequestHostPermissions()
	}
	p, err := native.StartProcess(ctx, c.HelperPath, c.HelperArgs, helperEnvironment(c.HelperEnv), native.NewCodec(uint32(c.MaxFrameBytes)))
	if err != nil {
		return nil, err
	}
	b := &Backend{process: p, config: c, operation: make(chan struct{}, 1), images: map[string]imageData{}, actions: map[string]struct{}{}}
	if _, err = b.Capabilities(ctx); err != nil {
		_ = b.Close(context.Background())
		return nil, err
	}
	return b, nil
}

func (b *Backend) request(ctx context.Context, command, sessionID, actionID string, payload map[string]any) (helperResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.config.RequestTimeout)
	defer cancel()
	deadline, _ := ctx.Deadline() // wall-clock deadline, including the caller's limit
	requestID := fmt.Sprintf("request-%d", b.requestCounter.Add(1))
	if sessionID == "" {
		sessionID = "host"
	}
	if actionID == "" {
		actionID = requestID
	}
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return helperResponse{}, &rejection{"invalid_payload"}
	}
	response, err := b.process.Call(ctx, native.Envelope{ProtocolVersion: native.ProtocolVersion, RequestID: requestID, SessionID: sessionID, ActionID: actionID, Deadline: deadline, Command: command, Payload: data})
	if err != nil {
		return helperResponse{}, err
	}
	var result helperResponse
	if json.Unmarshal(response.Payload, &result) != nil || result.OK == nil {
		b.process.Abort(errors.New("invalid helper response"))
		return result, errors.New("invalid helper response")
	}
	switch result.Outcome {
	case cu.OutcomeExecuted:
		if !*result.OK {
			err = errors.New("invalid helper success")
		}
	case cu.OutcomeRejected, cu.OutcomeNotStarted:
		if *result.OK {
			err = errors.New("invalid helper rejection")
		} else {
			return result, &rejection{"not_dispatched"}
		}
	case cu.OutcomeUnknown, cu.OutcomeFailed:
		return result, errors.New("helper could not confirm action outcome")
	default:
		err = errors.New("invalid helper outcome")
	}
	if err != nil {
		b.process.Abort(err)
	}
	return result, err
}

func (b *Backend) requestPermissions(ctx context.Context) error {
	// Permission state can change while the helper is alive after the user
	// toggles a macOS Privacy setting. Keep this probe idempotent but do not
	// cache it: every capabilities refresh must re-run the native request so
	// the UI can become ready without forcing an app restart.
	if _, err := b.request(ctx, commandRequestPermissions, "", "", nil); err != nil {
		return err
	}
	return nil
}

func (b *Backend) Capabilities(ctx context.Context) (cu.Capabilities, error) {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		return cu.Capabilities{}, &rejection{"closed"}
	}
	if err := b.requestPermissions(ctx); err != nil {
		return cu.Capabilities{}, err
	}
	response, err := b.request(ctx, commandReadiness, "", "", nil)
	if err != nil {
		return cu.Capabilities{}, err
	}
	data, _ := json.Marshal(response.Result)
	var caps cu.Capabilities
	if err = json.Unmarshal(data, &caps); err != nil {
		return caps, errors.New("invalid helper capabilities")
	}
	caps.ProtocolVersion = cu.ProtocolVersion
	caps.Platform = cu.PlatformMacOS
	caps.Backend = cu.BackendNativeHost
	caps.Actions = []cu.ActionKind{cu.ActionClick, cu.ActionDoubleClick, cu.ActionRightClick, cu.ActionMove, cu.ActionType, cu.ActionKey, cu.ActionHotkey, cu.ActionScroll, cu.ActionWait}
	caps.CoordinateSpace.Origin = cu.OriginTopLeft
	caps.CoordinateSpace.Unit = cu.CoordinatePixels
	if err = caps.Validate(); err != nil {
		return cu.Capabilities{}, err
	}
	b.mu.Lock()
	b.capabilities = caps
	b.mu.Unlock()
	return caps, nil
}

func (b *Backend) acquire(ctx context.Context) (uint64, error) {
	select {
	case b.operation <- struct{}{}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.stopped || b.paused {
		<-b.operation
		return 0, &rejection{"inactive"}
	}
	if err := ctx.Err(); err != nil {
		<-b.operation
		return 0, err
	}
	return b.epoch, nil
}
func (b *Backend) validEpoch(epoch uint64) bool {
	return !b.closed && !b.stopped && !b.paused && b.epoch == epoch
}

func (b *Backend) Observe(ctx context.Context, req cu.ObserveRequest) (cu.Observation, error) {
	epoch, err := b.acquire(ctx)
	if err != nil {
		return cu.Observation{}, err
	}
	defer func() { <-b.operation }()
	if req.SessionID == "" || req.WindowID != "" {
		return cu.Observation{}, &rejection{"unsupported_target"}
	}
	id := fmt.Sprintf("observation-%d", b.requestCounter.Add(1))
	response, err := b.request(ctx, commandObserve, req.SessionID, "", map[string]any{"display_id": req.DisplayID, "window_id": req.WindowID, "observation_id": id, "generation": epoch})
	if err != nil {
		return cu.Observation{}, err
	}
	data, mediaType, width, height, err := decodeImage(response.Result)
	if err != nil {
		return cu.Observation{}, err
	}
	display, ok := response.Result["display_id"].(string)
	scale, okScale := response.Result["scale_factor"].(float64)
	if !ok || display == "" || !okScale || math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 || scale > 8 || (req.DisplayID != "" && req.DisplayID != display) {
		return cu.Observation{}, errors.New("invalid screenshot coordinate metadata")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.validEpoch(epoch) {
		return cu.Observation{}, &rejection{"inactive"}
	}
	now := b.config.Now()
	// Use the helper's actual snapshot expiry, not this capture RPC's timeout.
	// Requiring its bounded metadata fails closed with stale/unmatched helpers.
	expiryText, okExpiry := response.Result["observation_expires_at"].(string)
	expires, expiryErr := time.Parse(time.RFC3339Nano, expiryText)
	if !okExpiry || expiryErr != nil || !expires.After(now) || expires.After(now.Add(cu.DefaultObservationTTL)) {
		return cu.Observation{}, errors.New("invalid observation expiration metadata")
	}
	caps := b.capabilities
	caps.CoordinateSpace = cu.CoordinateSpace{DisplayID: display, Origin: cu.OriginTopLeft, Unit: cu.CoordinatePixels, Width: width, Height: height, ScaleFactor: scale}
	obs := cu.Observation{ID: id, SessionID: req.SessionID, DisplayID: display, Width: width, Height: height, ScaleFactor: scale, Screenshot: cu.NewMediaRef(id, mediaType, data, width, height), Capabilities: caps, ObservedAt: now, ExpiresAt: expires}
	b.images = map[string]imageData{id: {data: data, mediaType: mediaType}}
	b.observation = obs
	b.capabilities = caps
	return obs, nil
}

func (b *Backend) ObservationImage(ctx context.Context, id string) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	image, ok := b.images[id]
	if !ok {
		return nil, "", errors.New("observation image unavailable")
	}
	return append([]byte(nil), image.data...), image.mediaType, nil
}

func (b *Backend) Execute(ctx context.Context, action cu.Action) (cu.ActionReceipt, error) {
	started := b.config.Now()
	receipt := cu.ActionReceipt{ActionID: action.ID, SessionID: action.SessionID, Platform: cu.PlatformMacOS, Backend: cu.BackendNativeHost, BeforeObservationID: action.ObservationID, Outcome: cu.OutcomeRejected, Verification: cu.VerificationNotChecked, RedactedActionSummary: action.RedactedSummary()}
	finish := func(err error) (cu.ActionReceipt, error) {
		receipt.CompletedAt = b.config.Now()
		receipt.Duration = receipt.CompletedAt.Sub(started)
		if err != nil {
			receipt.ErrorMessage = "computer action did not complete"
		}
		return receipt, err
	}
	epoch, err := b.acquire(ctx)
	if err != nil {
		return finish(err)
	}
	defer func() { <-b.operation }()
	b.mu.Lock()
	obs := b.observation
	_, duplicate := b.actions[action.ID]
	valid := b.validEpoch(epoch) && !duplicate && len(b.actions) < maxActions && action.SessionID == obs.SessionID && action.ObservationID == obs.ID && obs.ID != "" && (action.DisplayID == "" || action.DisplayID == obs.DisplayID) && action.WindowID == "" && action.Button == ""
	if valid {
		b.actions[action.ID] = struct{}{}
	}
	b.mu.Unlock()
	if !valid {
		return finish(&rejection{"invalid_action_binding"})
	}
	if err = action.Validate(b.config.Now(), obs); err != nil {
		return finish(&rejection{"invalid_action"})
	}
	b.mu.Lock()
	b.observation = cu.Observation{} // consume before dispatch, including rejection
	b.mu.Unlock()
	payload := map[string]any{"generation": epoch, "kind": action.Kind, "observation_id": action.ObservationID, "display_id": obs.DisplayID, "window_id": action.WindowID, "text": action.Text, "key": action.Key, "keys": action.Keys, "delta_x": action.DeltaX, "delta_y": action.DeltaY, "duration_ms": action.DurationMS}
	if action.Point != nil {
		payload["x"] = action.Point.X
		payload["y"] = action.Point.Y
	}
	response, err := b.request(ctx, commandExecute, action.SessionID, action.ID, payload)
	if err != nil {
		var rejected *rejection
		var call *native.CallError
		if !errors.As(err, &rejected) && !(errors.As(err, &call) && !call.MayHaveRun) {
			receipt.Outcome = cu.OutcomeUnknown
			receipt.Verification = cu.VerificationUnknown
			b.invalidate()
		}
		return finish(err)
	}
	data, mediaType, width, height, err := decodeImage(response.Result)
	if err != nil || width != obs.Width || height != obs.Height || response.Result["display_id"] != obs.DisplayID || response.Result["scale_factor"] != obs.ScaleFactor {
		receipt.Outcome = cu.OutcomeUnknown
		receipt.Verification = cu.VerificationUnknown
		b.invalidate()
		return finish(errors.New("invalid after screenshot"))
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.validEpoch(epoch) {
		receipt.Outcome = cu.OutcomeUnknown
		receipt.Verification = cu.VerificationUnknown
		return finish(errors.New("action interrupted"))
	}
	afterID := fmt.Sprintf("observation-after-%d", b.requestCounter.Add(1))
	b.images[afterID] = imageData{data: data, mediaType: mediaType}
	b.observation = cu.Observation{} // fresh observe required before another action
	receipt.Outcome = cu.OutcomeExecuted
	receipt.AfterObservationID = afterID
	after := cu.NewMediaRef(afterID, mediaType, data, width, height)
	receipt.After = &after
	return finish(nil)
}

func (b *Backend) invalidate() {
	b.mu.Lock()
	b.paused = true
	b.epoch++
	b.observation = cu.Observation{}
	b.mu.Unlock()
	b.process.Abort(errors.New("action outcome unknown; new helper required"))
}
func (b *Backend) Pause(ctx context.Context) error  { return b.control(ctx, commandPause) }
func (b *Backend) Resume(ctx context.Context) error { return b.control(ctx, commandResume) }
func (b *Backend) Stop(ctx context.Context) error   { return b.control(ctx, commandStop) }
func (b *Backend) control(ctx context.Context, command string) error {
	b.mu.Lock()
	if b.closed || (b.stopped && command != commandStop) {
		b.mu.Unlock()
		return &rejection{"inactive"}
	}
	if command == commandStop && b.stopped {
		b.mu.Unlock()
		return nil
	}
	b.epoch++
	epoch := b.epoch
	b.paused = true
	b.observation = cu.Observation{}
	if command == commandStop {
		b.stopped = true
		b.images = map[string]imageData{}
	}
	b.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, native.ShutdownGrace)
	defer cancel()
	_, err := b.request(ctx, command, "", "", map[string]any{"generation": epoch})
	b.mu.Lock()
	if command == commandResume && err == nil && b.epoch == epoch && !b.stopped && !b.closed {
		b.paused = false
	}
	b.mu.Unlock()
	if err != nil {
		b.process.Abort(errors.New("helper control failed"))
	}
	return err
}
func (b *Backend) Close(ctx context.Context) error {
	b.mu.Lock()
	first := !b.closed
	b.closed = true
	b.stopped = true
	b.paused = true
	b.epoch++
	b.images = map[string]imageData{}
	b.observation = cu.Observation{}
	b.mu.Unlock()
	if first {
		closeCtx, cancel := context.WithTimeout(ctx, native.ShutdownGrace)
		_, err := b.request(closeCtx, commandShutdown, "", "", nil)
		cancel()
		if err != nil {
			b.process.Abort(errors.New("helper shutdown failed"))
		}
	}
	return b.process.Wait(ctx)
}

var _ cu.Backend = (*Backend)(nil)
var _ interface {
	ObservationImage(context.Context, string) ([]byte, string, error)
} = (*Backend)(nil)
