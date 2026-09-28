// Package computerbridge proxies a trusted host computer session over an authenticated Unix socket.
// Session creation remains owned by the host coordinator; the bridge never grants OS permissions.
package computerbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const (
	CommandPath           = "/command"
	AuthorizationHeader   = "Authorization"
	BearerPrefix          = "Bearer "
	MaxRequestBytes       = 32 << 10
	MaxResponseBytes      = 24 << 20
	MaxImageBytes         = 16 << 20
	MaxImagePixels        = 32 << 20
	DefaultRequestTimeout = 15 * time.Second

	OpLookup       = "lookup"
	OpEnsure       = "ensure"
	OpCapabilities = "capabilities"
	OpObserve      = "observe"
	OpExecute      = "execute"
	OpPause        = "pause"
	OpResume       = "resume"
	OpStop         = "stop"
	OpImage        = "image"
)

// Errors never contain host error text, response bodies, tokens, or image data.
var (
	ErrInvalidConfig    = errors.New("computer bridge: invalid configuration")
	ErrInvalidRequest   = errors.New("computer bridge: invalid request")
	ErrRequestTooLarge  = errors.New("computer bridge: request exceeds limit")
	ErrTransport        = errors.New("computer bridge: request failed")
	ErrResponseTooLarge = errors.New("computer bridge: response exceeds limit")
	ErrInvalidResponse  = errors.New("computer bridge: invalid response")
	ErrRemote           = errors.New("computer bridge: host operation failed")
	ErrInvalidImage     = errors.New("computer bridge: invalid image")
)

// Service adds ephemeral image retrieval to the application service boundary.
// Implementations must not retain raw images in observations, logs or errors.
type Service interface {
	cu.Service
	ObservationImage(context.Context, cu.SessionOwner, string, string) ([]byte, string, error)
}

// SessionLookup only discovers sessions approved and bound by the host UI.
// A lookup must never create, approve, resume or reassign a session.
type SessionLookup interface {
	Lookup(context.Context, cu.SessionOwner) (string, error)
}

// SessionCoordinator lazily creates a host-owned session after the normal
// ComputerUse permission gate. It cannot mint macOS TCC approval.
type SessionCoordinator interface {
	EnsureComputerSession(context.Context, cu.SessionOwner) (string, error)
}

// Request is the host wire contract. Exactly one operation is sent per POST.
// The host must independently authenticate and authorize every request. Owner's
// SessionID is a conversation ID, distinct from the host SessionID below.
type Request struct {
	Op             string             `json:"op"`
	Owner          cu.SessionOwner    `json:"owner"`
	SessionID      string             `json:"session_id,omitempty"`
	ObserveRequest *cu.ObserveRequest `json:"observe_request,omitempty"`
	Action         *cu.Action         `json:"action,omitempty"`
	ObservationID  string             `json:"observation_id,omitempty"`
}

// Response wraps one operation's data, or a nonempty error. Only Execute may
// return both data and error, to preserve a validated, bound failure receipt.
// Error text is untrusted and is never propagated to callers.
type Response struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// SessionResponse acknowledges controls or returns a host-owned session ID.
type SessionResponse struct {
	SessionID string `json:"session_id"`
}

// LookupResponse identifies an existing host-approved session.
type LookupResponse = SessionResponse

// ImageResponse carries base64-encoded PNG bytes, not a data URL. Explicit IDs
// bind the bytes to the requested observation and approved host session.
type ImageResponse struct {
	SessionID     string `json:"session_id"`
	ObservationID string `json:"observation_id"`
	ImageData     string `json:"image_data"`
	MediaType     string `json:"media_type"`
}
