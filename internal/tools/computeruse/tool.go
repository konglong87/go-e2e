// Package computeruse adapts the provider-neutral Computer Use service to the
// agent Tool interface. It contains no platform or native helper code.
package computeruse

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"strings"

	"github.com/konglong87/go-e2e/internal/anthropic"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
	"github.com/konglong87/go-e2e/internal/tools"
)

const (
	ToolName     = "ComputerUse"
	pngMediaType = "image/png"
	maxPNGBytes  = 16 << 20
	maxPNGPixels = 32 << 20

	observationScreenshotGuidance = "Fresh Computer Use observation screenshot. Reference this observation ID for one input action. Each successful input returns another fresh observation; inspect it before deciding the next action."
	actionEvidenceGuidance        = "Computer Use post-action evidence only; not an actionable observation. On a successful action, call observe before the next input; never use this evidence image ID as observation_id. On an error or unknown outcome, stop without replaying input."
	screenshotCoordinateGuidance  = "Use full-image pixel coordinates with top-left origin, not window-relative or scaled-preview coordinates. Do not divide by scale_factor; the native backend performs that conversion."
)

type ObservationImageProvider interface {
	ObservationImage(context.Context, cu.SessionOwner, string, string) ([]byte, string, error)
}

// Tool carries no authority: registry clones may safely share this value.
type Tool struct{}

func New() Tool { return Tool{} }

func (Tool) Name() string { return ToolName }

func (Tool) Description() string {
	return "Observe and operate an explicitly approved computer session using structured actions. Each input action consumes a fresh observation. A successful input returns its receipt plus a new observation and screenshot for the next decision. Use observation.id, never receipt.after_observation_id. Observe again if the UI has not settled. Stop on errors or unknown outcomes; never replay input."
}

func (Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string"},"action":{"type":"string","enum":["observe","click","double_click","right_click","move","type","key","hotkey","scroll","wait","pause","stop"]},"display_id":{"type":"string"},"window_id":{"type":"string"},"observation_id":{"type":"string"},"x":{"type":"integer"},"y":{"type":"integer"},"text":{"type":"string"},"key":{"type":"string"},"keys":{"type":"array","items":{"type":"string"}},"delta_x":{"type":"integer"},"delta_y":{"type":"integer"},"duration_ms":{"type":"integer"}},"required":["session_id","action"],"additionalProperties":false}`)
}

func (Tool) ExecutionPolicy() tools.ExecutionPolicy {
	return tools.ExecutionPolicy{Concurrency: tools.ConcurrencySerial}
}

func (t Tool) Run(ctx context.Context, input json.RawMessage, tc tools.Context) tools.Result {
	if tc.SubagentDepth > 0 {
		return errorResult("capability_unavailable", "computer use is not available to subagents")
	}
	service := tc.ComputerUse
	if service == nil {
		return errorResult("capability_unavailable", "computer use service is unavailable")
	}
	if !tc.ComputerUseImageSupported {
		return errorResult("capability_unavailable", "computer use requires provider image input support")
	}
	if tc.TenantID == 0 || tc.UserID == 0 || tc.SessionID == 0 {
		return errorResult("invalid_context", "trusted tenant, user, and session context are required")
	}
	var params request
	if err := decode(input, &params); err != nil {
		return errorResult("invalid_input", "invalid computer use request")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return errorResult("invalid_input", "session_id is required")
	}
	owner := cu.SessionOwner{TenantID: tc.TenantID, UserID: tc.UserID, SessionID: tc.SessionID}
	switch cu.ActionKind(params.Action) {
	case cu.ActionObserve:
		return t.observe(ctx, service, owner, params)
	case cu.ActionPause:
		return t.control(ctx, owner, params, func() error { return service.Pause(ctx, owner, params.SessionID) })
	case cu.ActionStop:
		return t.control(ctx, owner, params, func() error { return service.Stop(ctx, owner, params.SessionID) })
	case cu.ActionClick, cu.ActionDoubleClick, cu.ActionRightClick, cu.ActionMove, cu.ActionType, cu.ActionKey, cu.ActionHotkey, cu.ActionScroll, cu.ActionWait:
		return t.execute(ctx, service, owner, params, tc)
	default:
		return errorResult("invalid_input", "unsupported computer action")
	}
}

type request struct {
	SessionID     string   `json:"session_id"`
	Action        string   `json:"action"`
	DisplayID     string   `json:"display_id"`
	WindowID      string   `json:"window_id"`
	ObservationID string   `json:"observation_id"`
	X             *int     `json:"x"`
	Y             *int     `json:"y"`
	Text          string   `json:"text"`
	Key           string   `json:"key"`
	Keys          []string `json:"keys"`
	DeltaX        int      `json:"delta_x"`
	DeltaY        int      `json:"delta_y"`
	DurationMS    int      `json:"duration_ms"`
}

func (t Tool) observe(ctx context.Context, service cu.Service, owner cu.SessionOwner, params request) tools.Result {
	_, result := t.captureObservation(ctx, service, owner, params)
	return result
}

func (t Tool) captureObservation(ctx context.Context, service cu.Service, owner cu.SessionOwner, params request) (cu.Observation, tools.Result) {
	observation, err := service.Observe(ctx, owner, cu.ObserveRequest{SessionID: params.SessionID, DisplayID: params.DisplayID, WindowID: params.WindowID})
	if err != nil {
		return cu.Observation{}, errorResult("observe_failed", "computer observation failed")
	}
	messages, err := observationImage(ctx, service, owner, params.SessionID, observation.ID, observationScreenshotGuidance)
	if err != nil {
		return cu.Observation{}, errorResult("observation_image_failed", "valid computer screenshot unavailable")
	}
	return observation, tools.Result{Content: marshal(map[string]any{"observation": observation}), ContextMessages: messages}
}

// Bound both compressed bytes and decoded pixels before allocating the image.
// A MIME label or PNG signature alone does not establish a valid screenshot.
func observationImage(ctx context.Context, service cu.Service, owner cu.SessionOwner, sessionID, observationID, guidance string) ([]anthropic.MessageParam, error) {
	provider, ok := service.(ObservationImageProvider)
	if !ok || strings.TrimSpace(observationID) == "" {
		return nil, errors.New("screenshot unavailable")
	}
	data, mediaType, err := provider.ObservationImage(ctx, owner, sessionID, observationID)
	if err != nil || mediaType != pngMediaType || len(data) == 0 || len(data) > maxPNGBytes {
		return nil, errors.New("invalid screenshot")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > maxPNGPixels/config.Height {
		return nil, errors.New("invalid screenshot dimensions")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return nil, errors.New("invalid screenshot content")
	}
	return []anthropic.MessageParam{{Role: "user", Content: []anthropic.ContentBlock{
		{Type: "text", Text: fmt.Sprintf("%s Image ID: %q. Image dimensions: %dx%d pixels. %s", guidance, observationID, config.Width, config.Height, screenshotCoordinateGuidance)},
		{Type: "image", Source: &anthropic.ContentSource{Type: "base64", MediaType: pngMediaType, Data: base64.StdEncoding.EncodeToString(data)}},
	}}}, nil
}

// Run and tool-use IDs are mandatory: neither model IDs nor a session/action
// fallback can distinguish invocations. Length-delimited JSON also avoids
// separator collisions. Include owner and batch to namespace runtime IDs.
func invocationActionID(tc tools.Context) (string, error) {
	if strings.TrimSpace(tc.Invocation.RunID) == "" || strings.TrimSpace(tc.Invocation.ToolUseID) == "" {
		return "", errors.New("trusted runtime invocation is required")
	}
	data, err := json.Marshal(struct {
		Owner      cu.SessionOwner
		Invocation tools.Invocation
	}{cu.SessionOwner{TenantID: tc.TenantID, UserID: tc.UserID, SessionID: tc.SessionID}, tc.Invocation})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("computer-%x", sha256.Sum256(data)), nil
}

func (t Tool) execute(ctx context.Context, service cu.Service, owner cu.SessionOwner, params request, tc tools.Context) tools.Result {
	id, err := invocationActionID(tc)
	if err != nil {
		return errorResult("invalid_context", "trusted runtime invocation is required")
	}
	action := cu.Action{
		ID: id, SessionID: params.SessionID, ObservationID: params.ObservationID, Kind: cu.ActionKind(params.Action),
		DisplayID: params.DisplayID, WindowID: params.WindowID, Text: params.Text, Key: params.Key,
		Keys: params.Keys, DeltaX: params.DeltaX, DeltaY: params.DeltaY, DurationMS: params.DurationMS,
	}
	if params.X != nil && params.Y != nil {
		action.Point = &cu.Point{X: *params.X, Y: *params.Y}
	}
	receipt, executeErr := service.Execute(ctx, owner, action)
	// Errors can accompany an unknown outcome after input was dispatched. Never
	// discard that receipt or suggest the caller should replay the action.
	receipt.ErrorMessage = ""
	if receipt.ErrorCode != "" {
		receipt.ErrorCode = "action_failed"
	}
	receipt.RedactedActionSummary = string(action.Kind)
	payload := map[string]any{"receipt": receipt}
	out := tools.Result{IsError: executeErr != nil || receipt.Outcome != cu.OutcomeExecuted}
	if out.IsError {
		payload["error_code"] = "action_failed"
		payload["message"] = "computer action failed; inspect receipt before any further action"
	}
	if receipt.AfterObservationID != "" {
		messages, imageErr := observationImage(ctx, service, owner, params.SessionID, receipt.AfterObservationID, actionEvidenceGuidance)
		if imageErr != nil {
			out.IsError = true
			payload["image_error_code"] = "observation_image_failed"
		} else {
			out.ContextMessages = messages
		}
	} else {
		out.IsError = true
		payload["image_error_code"] = "observation_image_failed"
	}
	// A receipt's image is never authority for the next input. Compose a genuine
	// owner-bound Observe only after unambiguous success and valid visual evidence.
	// This reuses Controller/native focus, permission, cancellation and TTL gates;
	// it never retries input or silently promotes an evidence ID to an observation.
	if !out.IsError {
		if ctx.Err() != nil {
			out.IsError = true
			payload["error_code"] = "post_action_observation_failed"
		} else {
			observation, fresh := t.captureObservation(ctx, service, owner, params)
			if fresh.IsError {
				out.IsError = true
				payload["error_code"] = "post_action_observation_failed"
			} else {
				payload["observation"] = observation
				out.ContextMessages = fresh.ContextMessages
			}
		}
		if out.IsError {
			payload["message"] = "input was executed but fresh observation is unavailable; stop without replaying input"
		}
	}
	out.Content = marshal(payload)
	return out
}

func (t Tool) control(ctx context.Context, owner cu.SessionOwner, params request, fn func() error) tools.Result {
	if err := fn(); err != nil {
		return errorResult("control_failed", "computer session control failed")
	}
	return tools.Result{Content: marshal(map[string]string{"session_id": params.SessionID, "action": params.Action, "status": "ok"})}
}

func decode(input json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("unexpected trailing JSON")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func marshal(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return `{"error_code":"internal_error"}`
	}
	return string(payload)
}

func errorResult(code, message string) tools.Result {
	return tools.Result{Content: marshal(map[string]string{"error_code": code, "message": message}), IsError: true}
}
