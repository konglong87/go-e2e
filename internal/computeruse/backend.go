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

// BackendLauncher is the compatibility boundary used by a trusted host
// backend. Controller launch calls resolve TargetID through TargetRegistry
// before passing this opaque provider key to the backend.
type BackendLauncher interface {
	LaunchApp(context.Context, string) (LaunchReceipt, error)
}

// BackendApplicationLauncher is retained as a source-compatible alias.
type BackendApplicationLauncher = BackendLauncher
