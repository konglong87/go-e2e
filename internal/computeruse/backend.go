package computeruse

import "context"

// ObserveRequest keeps observation inputs platform-neutral. A backend may use
// the requested display/window as a hint, but it must return the actual target
// in Observation and Capabilities.
type ObserveRequest struct {
	SessionID string `json:"session_id"`
	DisplayID string `json:"display_id,omitempty"`
	WindowID  string `json:"window_id,omitempty"`
}

// Backend is the single execution boundary for screenshots and host input.
// Implementations must not accept raw model prompts, provider credentials, or
// unvalidated action payloads.
type Backend interface {
	Capabilities(context.Context) (Capabilities, error)
	Observe(context.Context, ObserveRequest) (Observation, error)
	Execute(context.Context, Action) (ActionReceipt, error)
	Pause(context.Context) error
	Resume(context.Context) error
	Stop(context.Context) error
	Close(context.Context) error
}

// ApplicationLauncher is an optional trusted host capability. Launching is a
// session-management operation, not a model-generated input event, and must
// be implemented by an allowlisted native host launcher.
type BackendApplicationLauncher interface {
	LaunchApp(context.Context, string) (LaunchReceipt, error)
}
