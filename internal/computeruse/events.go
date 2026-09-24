package computeruse

type EventKind string

const (
	EventSessionStarted  EventKind = "computer_session_started"
	EventObservation     EventKind = "computer_observation"
	EventActionRequested EventKind = "computer_action_requested"
	EventActionFinished  EventKind = "computer_action_finished"
	EventPermission      EventKind = "computer_permission"
	EventPaused          EventKind = "computer_paused"
	EventResumed         EventKind = "computer_resumed"
	EventStopped         EventKind = "computer_stopped"
	EventFailed          EventKind = "computer_failed"
)

type Event struct {
	Kind          EventKind      `json:"kind"`
	SessionID     string         `json:"session_id"`
	ActionSummary string         `json:"redacted_action_summary,omitempty"`
	Receipt       *ActionReceipt `json:"receipt,omitempty"`
	State         SessionState   `json:"state,omitempty"`
}
