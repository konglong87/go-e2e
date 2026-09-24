// Package computeruse adapts the provider-neutral Computer Use service to the
// agent Tool interface. It contains no platform or native helper code.
package computeruse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/konglong87/go-e2e/internal/anthropic"
	cu "github.com/konglong87/go-e2e/internal/computeruse"
	"github.com/konglong87/go-e2e/internal/tools"
)

const ToolName = "ComputerUse"

type ObservationImageProvider interface {
	ObservationImage(context.Context, cu.SessionOwner, string, string) ([]byte, string, error)
}

type Tool struct {
	service cu.Service
}

func New(service cu.Service) Tool { return Tool{service: service} }

func (Tool) Name() string { return ToolName }

func (Tool) Description() string {
	return "Observe and operate an explicitly approved computer session using structured actions. Each input action must reference the latest observation."
}

func (Tool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string"},"action":{"type":"string","enum":["observe","click","double_click","right_click","move","type","key","hotkey","scroll","wait","pause","resume","stop"]},"display_id":{"type":"string"},"window_id":{"type":"string"},"observation_id":{"type":"string"},"action_id":{"type":"string"},"x":{"type":"integer"},"y":{"type":"integer"},"text":{"type":"string"},"key":{"type":"string"},"keys":{"type":"array","items":{"type":"string"}},"delta_x":{"type":"integer"},"delta_y":{"type":"integer"},"duration_ms":{"type":"integer"}},"required":["action"],"additionalProperties":false}`)
}

func (Tool) ExecutionPolicy() tools.ExecutionPolicy {
	return tools.ExecutionPolicy{Concurrency: tools.ConcurrencySerial}
}

func (t Tool) Run(ctx context.Context, input json.RawMessage, tc tools.Context) tools.Result {
	if t.service == nil {
		return errorResult("capability_unavailable", "computer use service is unavailable")
	}
	if tc.ComputerUseImageSupported == false {
		return errorResult("capability_unavailable", "computer use requires provider image input support")
	}
	if tc.TenantID == 0 || tc.UserID == 0 || tc.SessionID == 0 {
		return errorResult("invalid_context", "trusted tenant, user, and session context are required")
	}
	var params request
	if err := decode(input, &params); err != nil {
		return errorResult("invalid_input", err.Error())
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return errorResult("invalid_input", "session_id is required")
	}
	owner := cu.SessionOwner{TenantID: tc.TenantID, UserID: tc.UserID}
	switch cu.ActionKind(params.Action) {
	case cu.ActionObserve:
		return t.observe(ctx, owner, params)
	case cu.ActionPause:
		return t.control(ctx, owner, params, func() error { return t.service.Pause(ctx, owner, params.SessionID) })
	case cu.ActionResume:
		return t.control(ctx, owner, params, func() error { return t.service.Resume(ctx, owner, params.SessionID) })
	case cu.ActionStop:
		return t.control(ctx, owner, params, func() error { return t.service.Stop(ctx, owner, params.SessionID) })
	case cu.ActionClick, cu.ActionDoubleClick, cu.ActionRightClick, cu.ActionMove, cu.ActionType, cu.ActionKey, cu.ActionHotkey, cu.ActionScroll, cu.ActionWait:
		return t.execute(ctx, owner, params, tc)
	default:
		return errorResult("invalid_input", fmt.Sprintf("unsupported computer action %q", params.Action))
	}
}

type request struct {
	SessionID     string   `json:"session_id"`
	Action        string   `json:"action"`
	DisplayID     string   `json:"display_id"`
	WindowID      string   `json:"window_id"`
	ObservationID string   `json:"observation_id"`
	ActionID      string   `json:"action_id"`
	X             *int     `json:"x"`
	Y             *int     `json:"y"`
	Text          string   `json:"text"`
	Key           string   `json:"key"`
	Keys          []string `json:"keys"`
	DeltaX        int      `json:"delta_x"`
	DeltaY        int      `json:"delta_y"`
	DurationMS    int      `json:"duration_ms"`
}

func (t Tool) observe(ctx context.Context, owner cu.SessionOwner, params request) tools.Result {
	observation, err := t.service.Observe(ctx, owner, cu.ObserveRequest{SessionID: params.SessionID, DisplayID: params.DisplayID, WindowID: params.WindowID})
	if err != nil {
		return errorResult("observe_failed", err.Error())
	}
	result := map[string]any{"observation": observation}
	out := tools.Result{Content: marshal(result)}
	provider, ok := t.service.(ObservationImageProvider)
	if !ok {
		return errorResult("observation_image_unavailable", "computer use service cannot provide screenshot content")
	}
	data, mediaType, imageErr := provider.ObservationImage(ctx, owner, params.SessionID, observation.ID)
	if imageErr != nil {
		return errorResult("observation_image_failed", imageErr.Error())
	}
	if len(data) == 0 || mediaType == "" {
		return errorResult("observation_image_missing", "observation did not return image content")
	}
	out.ContextMessages = []anthropic.MessageParam{{Role: "user", Content: []anthropic.ContentBlock{
		{Type: "text", Text: "Computer Use observation screenshot."},
		{Type: "image", Source: &anthropic.ContentSource{Type: "base64", MediaType: mediaType, Data: base64.StdEncoding.EncodeToString(data)}},
	}}}
	return out
}

func (t Tool) execute(ctx context.Context, owner cu.SessionOwner, params request, tc tools.Context) tools.Result {
	action := cu.Action{
		ID: "", SessionID: params.SessionID, ObservationID: params.ObservationID, Kind: cu.ActionKind(params.Action),
		DisplayID: params.DisplayID, WindowID: params.WindowID, Text: params.Text, Key: params.Key,
		Keys: params.Keys, DeltaX: params.DeltaX, DeltaY: params.DeltaY, DurationMS: params.DurationMS,
	}
	if params.X != nil && params.Y != nil {
		action.Point = &cu.Point{X: *params.X, Y: *params.Y}
	}
	if actionID := strings.TrimSpace(params.ActionID); actionID != "" {
		action.ID = actionID
	} else if toolUseID := strings.TrimSpace(tc.Invocation.ToolUseID); toolUseID != "" {
		action.ID = toolUseID
	} else {
		action.ID = params.SessionID + ":" + params.Action
	}
	receipt, err := t.service.Execute(ctx, owner, action)
	if err != nil {
		return errorResult("action_failed", err.Error())
	}
	return tools.Result{Content: marshal(map[string]any{"receipt": receipt})}
}

func (t Tool) control(ctx context.Context, owner cu.SessionOwner, params request, fn func() error) tools.Result {
	if err := fn(); err != nil {
		return errorResult("control_failed", err.Error())
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
