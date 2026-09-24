package computeruse

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
	"github.com/konglong87/go-e2e/internal/tools"
)

type serviceStub struct {
	observation cu.Observation
	receipt     cu.ActionReceipt
	image       []byte
	imageType   string
	lastAction  cu.Action
}

func (s *serviceStub) Capabilities(context.Context, cu.SessionOwner, string) (cu.Capabilities, error) {
	return s.observation.Capabilities, nil
}
func (s *serviceStub) Observe(_ context.Context, _ cu.SessionOwner, request cu.ObserveRequest) (cu.Observation, error) {
	out := s.observation
	out.SessionID = request.SessionID
	return out, nil
}
func (s *serviceStub) Execute(_ context.Context, _ cu.SessionOwner, action cu.Action) (cu.ActionReceipt, error) {
	s.lastAction = action
	out := s.receipt
	out.ActionID = action.ID
	out.SessionID = action.SessionID
	return out, nil
}
func (*serviceStub) Pause(context.Context, cu.SessionOwner, string) error  { return nil }
func (*serviceStub) Resume(context.Context, cu.SessionOwner, string) error { return nil }
func (*serviceStub) Stop(context.Context, cu.SessionOwner, string) error   { return nil }
func (s *serviceStub) ObservationImage(context.Context, cu.SessionOwner, string, string) ([]byte, string, error) {
	if len(s.image) == 0 {
		return nil, "", errors.New("missing image")
	}
	return s.image, s.imageType, nil
}

func testContext(service cu.Service) tools.Context {
	return tools.Context{TenantID: 7, UserID: 11, SessionID: 13, ComputerUse: service, ComputerUseImageSupported: true, Invocation: tools.Invocation{ToolUseID: "tool-1"}}
}

func TestObservePassesRealScreenshotImageContext(t *testing.T) {
	service := &serviceStub{observation: cu.Observation{ID: "obs-1", Width: 10, Height: 10, Capabilities: cu.Capabilities{ImageSupported: true}}, image: []byte("png"), imageType: "image/png"}
	result := New(service).Run(context.Background(), json.RawMessage(`{"session_id":"computer-1","action":"observe"}`), testContext(service))
	if result.IsError || len(result.ContextMessages) != 1 || len(result.ContextMessages[0].Content) != 2 {
		t.Fatalf("result = %+v", result)
	}
	image := result.ContextMessages[0].Content[1]
	if image.Type != "image" || image.Source == nil || image.Source.MediaType != "image/png" || image.Source.Data == "" {
		t.Fatalf("image context = %+v", image)
	}
}

func TestInputUsesTrustedInvocationIDAndDoesNotLeakText(t *testing.T) {
	service := &serviceStub{receipt: cu.ActionReceipt{Outcome: cu.OutcomeExecuted}}
	result := New(service).Run(context.Background(), json.RawMessage(`{"session_id":"computer-1","action":"type","observation_id":"obs-1","text":"password-123"}`), testContext(service))
	if result.IsError || service.lastAction.ID != "tool-1" {
		t.Fatalf("result = %+v action = %+v", result, service.lastAction)
	}
	if service.lastAction.Text != "password-123" {
		t.Fatalf("service did not receive structured action text: %+v", service.lastAction)
	}
	if strings.Contains(result.Content, service.lastAction.Text) {
		t.Fatalf("tool result leaked sensitive text: %s", result.Content)
	}
}

func TestRejectsUnknownInputAndMissingImageCapability(t *testing.T) {
	service := &serviceStub{}
	unknown := New(service).Run(context.Background(), json.RawMessage(`{"session_id":"computer-1","action":"observe","unknown":true}`), testContext(service))
	if !unknown.IsError || !strings.Contains(unknown.Content, "invalid_input") {
		t.Fatalf("unknown field result = %+v", unknown)
	}
	noImage := testContext(service)
	noImage.ComputerUseImageSupported = false
	result := New(service).Run(context.Background(), json.RawMessage(`{"session_id":"computer-1","action":"observe"}`), noImage)
	if !result.IsError || !strings.Contains(result.Content, "capability_unavailable") {
		t.Fatalf("no-image result = %+v", result)
	}
}

func TestToolIsSerial(t *testing.T) {
	if got := New(nil).ExecutionPolicy().Concurrency; got != tools.ConcurrencySerial {
		t.Fatalf("concurrency = %s", got)
	}
}
