package computerbridge

import (
	"context"
	"errors"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const (
	receiptHostErrorCode    = "host_operation_failed"
	receiptUnknownErrorCode = "bridge_outcome_unknown"
)

func (c *Client) Lookup(ctx context.Context, owner cu.SessionOwner) (string, error) {
	var out LookupResponse
	if err := c.data(ctx, Request{Op: OpLookup, Owner: owner}, &out); err != nil {
		return "", err
	}
	if !validID(out.SessionID) {
		return "", ErrInvalidResponse
	}
	return out.SessionID, nil
}

func (c *Client) EnsureComputerSession(ctx context.Context, owner cu.SessionOwner) (string, error) {
	var out SessionResponse
	if err := c.data(ctx, Request{Op: OpEnsure, Owner: owner}, &out); err != nil {
		return "", err
	}
	if !validID(out.SessionID) {
		return "", ErrInvalidResponse
	}
	return out.SessionID, nil
}

func (c *Client) Capabilities(ctx context.Context, owner cu.SessionOwner, sessionID string) (cu.Capabilities, error) {
	var out cu.Capabilities
	if err := c.data(ctx, Request{Op: OpCapabilities, Owner: owner, SessionID: sessionID}, &out); err != nil {
		return cu.Capabilities{}, err
	}
	if out.Validate() != nil {
		return cu.Capabilities{}, ErrInvalidResponse
	}
	return out, nil
}

func (c *Client) Observe(ctx context.Context, owner cu.SessionOwner, in cu.ObserveRequest) (cu.Observation, error) {
	var out cu.Observation
	if err := c.data(ctx, Request{Op: OpObserve, Owner: owner, SessionID: in.SessionID, ObserveRequest: &in}, &out); err != nil {
		return cu.Observation{}, err
	}
	if !validObservation(out, in) {
		return cu.Observation{}, ErrInvalidResponse
	}
	return out, nil
}

// Execute never retries. Once dispatch is attempted, an invalid or absent
// acknowledgement cannot establish whether host input occurred. Return a bound
// unknown receipt AND error so callers cannot mistake a zero receipt for safety.
func (c *Client) Execute(ctx context.Context, owner cu.SessionOwner, action cu.Action) (cu.ActionReceipt, error) {
	response, attempted, err := c.call(ctx, Request{Op: OpExecute, Owner: owner, SessionID: action.SessionID, Action: &action})
	if !attempted {
		return cu.ActionReceipt{}, err
	}
	if err != nil && !errors.Is(err, ErrRemote) {
		return unknownReceipt(action), err
	}
	var receipt cu.ActionReceipt
	if decodeStrict(response.Data, &receipt) != nil || !validReceipt(receipt, action) {
		if err == nil {
			err = ErrInvalidResponse
		}
		return unknownReceipt(action), err
	}
	// A simultaneous success receipt and error is not an execution guarantee.
	if receipt.Outcome == cu.OutcomeExecuted && (err != nil || receipt.ErrorCode != "" || receipt.ErrorMessage != "") {
		return unknownReceipt(action), ErrInvalidResponse
	}
	receipt.ErrorMessage = ""
	receipt.RedactedActionSummary = action.RedactedSummary()
	if receipt.ErrorCode != "" {
		receipt.ErrorCode = receiptHostErrorCode
	}
	if receipt.Outcome != cu.OutcomeExecuted {
		receipt.ErrorCode = receiptHostErrorCode
		return receipt, ErrRemote
	}
	return receipt, nil
}

func unknownReceipt(a cu.Action) cu.ActionReceipt {
	return cu.ActionReceipt{
		ActionID: a.ID, SessionID: a.SessionID, BeforeObservationID: a.ObservationID,
		Outcome: cu.OutcomeUnknown, Verification: cu.VerificationUnknown,
		RedactedActionSummary: a.RedactedSummary(), ErrorCode: receiptUnknownErrorCode,
	}
}

func (c *Client) Pause(ctx context.Context, owner cu.SessionOwner, id string) error {
	return c.control(ctx, owner, id, OpPause)
}
func (c *Client) Resume(ctx context.Context, owner cu.SessionOwner, id string) error {
	return c.control(ctx, owner, id, OpResume)
}
func (c *Client) Stop(ctx context.Context, owner cu.SessionOwner, id string) error {
	return c.control(ctx, owner, id, OpStop)
}
func (c *Client) control(ctx context.Context, owner cu.SessionOwner, id, op string) error {
	var out SessionResponse
	if err := c.data(ctx, Request{Op: op, Owner: owner, SessionID: id}, &out); err != nil {
		return err
	}
	if out.SessionID != id {
		return ErrInvalidResponse
	}
	return nil
}

func (c *Client) data(ctx context.Context, request Request, out any) error {
	response, _, err := c.call(ctx, request)
	if err != nil {
		return err
	}
	return decodeStrict(response.Data, out)
}
