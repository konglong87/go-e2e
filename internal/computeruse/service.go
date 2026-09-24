package computeruse

import "context"

// Service is the trusted application boundary used by the agent tool and the
// Desktop control plane. Implementations own session lookup, tenant/user
// ownership checks, serial execution, and backend lifecycle.
type Service interface {
	Capabilities(context.Context, SessionOwner, string) (Capabilities, error)
	Observe(context.Context, SessionOwner, ObserveRequest) (Observation, error)
	Execute(context.Context, SessionOwner, Action) (ActionReceipt, error)
	Pause(context.Context, SessionOwner, string) error
	Resume(context.Context, SessionOwner, string) error
	Stop(context.Context, SessionOwner, string) error
}
