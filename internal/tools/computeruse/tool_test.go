package computeruse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
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
	owners      []cu.SessionOwner
	calls       int
	imageID     string
	serviceErr  error
	imageErr    error
}

func (s *serviceStub) record(owner cu.SessionOwner) { s.calls++; s.owners = append(s.owners, owner) }
func (s *serviceStub) Capabilities(_ context.Context, owner cu.SessionOwner, _ string) (cu.Capabilities, error) {
	s.record(owner)
	return s.observation.Capabilities, s.serviceErr
}
func (s *serviceStub) Observe(_ context.Context, owner cu.SessionOwner, request cu.ObserveRequest) (cu.Observation, error) {
	s.record(owner)
	out := s.observation
	out.SessionID = request.SessionID
	return out, s.serviceErr
}
func (s *serviceStub) Execute(_ context.Context, owner cu.SessionOwner, action cu.Action) (cu.ActionReceipt, error) {
	s.record(owner)
	s.lastAction = action
	out := s.receipt
	out.ActionID = action.ID
	out.SessionID = action.SessionID
	return out, s.serviceErr
}
func (s *serviceStub) Pause(_ context.Context, owner cu.SessionOwner, _ string) error {
	s.record(owner)
	return s.serviceErr
}
func (s *serviceStub) Resume(_ context.Context, owner cu.SessionOwner, _ string) error {
	s.record(owner)
	return s.serviceErr
}
func (s *serviceStub) Stop(_ context.Context, owner cu.SessionOwner, _ string) error {
	s.record(owner)
	return s.serviceErr
}
func (s *serviceStub) ObservationImage(_ context.Context, owner cu.SessionOwner, _, id string) ([]byte, string, error) {
	s.record(owner)
	s.imageID = id
	return s.image, s.imageType, s.imageErr
}
func testContext(service cu.Service) tools.Context {
	return tools.Context{TenantID: 7, UserID: 11, SessionID: 13, ComputerUse: service, ComputerUseImageSupported: true, Invocation: tools.Invocation{RunID: "run-1", ToolUseID: "tool-1"}}
}
func realPNG(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
func screenshotService(t *testing.T) *serviceStub {
	return &serviceStub{observation: cu.Observation{ID: "obs-1", Width: 10, Height: 10}, image: realPNG(t), imageType: pngMediaType}
}
func runRequest(service cu.Service, input string) tools.Result {
	return New().Run(context.Background(), json.RawMessage(input), testContext(service))
}
func assertImage(t *testing.T, result tools.Result, want []byte) {
	t.Helper()
	if len(result.ContextMessages) != 1 || len(result.ContextMessages[0].Content) != 2 {
		t.Fatalf("missing image: %+v", result)
	}
	block := result.ContextMessages[0].Content[1]
	if block.Type != "image" || block.Source == nil || block.Source.MediaType != pngMediaType {
		t.Fatalf("invalid block: %+v", block)
	}
	data, err := base64.StdEncoding.DecodeString(block.Source.Data)
	if err != nil || !bytes.Equal(data, want) {
		t.Fatalf("image mismatch: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
}
func TestObservePassesRealScreenshotImageContext(t *testing.T) {
	service := screenshotService(t)
	result := runRequest(service, `{"session_id":"computer-1","action":"observe"}`)
	if result.IsError {
		t.Fatal(result.Content)
	}
	assertImage(t, result, service.image)
	for _, owner := range service.owners {
		if owner != (cu.SessionOwner{TenantID: 7, UserID: 11, SessionID: 13}) {
			t.Fatalf("untrusted owner: %+v", owner)
		}
	}
}
func TestClonedToolUsesOnlyCurrentTrustedCapability(t *testing.T) {
	registry := tools.NewRegistry(New()).Clone()
	tool, ok := registry.Get(ToolName)
	if !ok {
		t.Fatal("missing tool")
	}
	first, second := screenshotService(t), screenshotService(t)
	input := json.RawMessage(`{"session_id":"computer-1","action":"observe"}`)
	for _, service := range []*serviceStub{first, second} {
		if result := tool.Run(context.Background(), input, testContext(service)); result.IsError {
			t.Fatal(result.Content)
		}
	}
	if first.calls != 2 || second.calls != 2 {
		t.Fatalf("wrong services: %d / %d", first.calls, second.calls)
	}
	if result := tool.Run(context.Background(), input, testContext(nil)); !result.IsError {
		t.Fatal("reused previous capability")
	}
	if first.calls != 2 || second.calls != 2 {
		t.Fatal("retained service called")
	}
}
func TestRejectsUntrustedContextBeforeServiceCall(t *testing.T) {
	cases := []struct {
		name   string
		change func(*tools.Context)
	}{
		{"subagent", func(tc *tools.Context) { tc.SubagentDepth = 1 }},
		{"deep subagent", func(tc *tools.Context) { tc.SubagentDepth = 5 }},
		{"no service", func(tc *tools.Context) { tc.ComputerUse = nil }},
		{"no image support", func(tc *tools.Context) { tc.ComputerUseImageSupported = false }},
		{"no tenant", func(tc *tools.Context) { tc.TenantID = 0 }},
		{"no user", func(tc *tools.Context) { tc.UserID = 0 }},
		{"no conversation", func(tc *tools.Context) { tc.SessionID = 0 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			service := screenshotService(t)
			tc := testContext(service)
			test.change(&tc)
			result := New().Run(context.Background(), json.RawMessage(`{"session_id":"computer-1","action":"observe"}`), tc)
			if !result.IsError || service.calls != 0 {
				t.Fatalf("not fail closed: %+v calls=%d", result, service.calls)
			}
		})
	}
}
func TestRejectsModelAuthorityAndMalformedInput(t *testing.T) {
	for _, input := range []string{
		`{"session_id":"computer-1","action":"resume"}`,
		`{"session_id":"computer-1","action":"start"}`,
		`{"session_id":"computer-1","action":"approve"}`,
		`{"session_id":"computer-1","action":"type","action_id":"model-id"}`,
		`{"session_id":"computer-1","action":"observe","unknown":true}`,
		`{"session_id":"computer-1","action":"observe"} {}`,
		`{"action":"observe"}`, `null`,
	} {
		t.Run(input, func(t *testing.T) {
			service := screenshotService(t)
			result := runRequest(service, input)
			if !result.IsError || service.calls != 0 {
				t.Fatalf("not rejected: %+v", result)
			}
		})
	}
	schema := string(New().InputSchema())
	for _, forbidden := range []string{`"resume"`, `"start"`, `"approve"`, `"action_id"`} {
		if strings.Contains(schema, forbidden) {
			t.Fatalf("model authority in schema: %s", forbidden)
		}
	}
}
func TestInvocationActionIDRequiredUniqueAndStable(t *testing.T) {
	service := &serviceStub{receipt: cu.ActionReceipt{Outcome: cu.OutcomeExecuted}}
	input := json.RawMessage(`{"session_id":"computer-1","action":"type","text":"password-123"}`)
	base := testContext(service)
	for _, invocation := range []tools.Invocation{{}, {RunID: "run-1"}, {ToolUseID: "tool-1"}} {
		tc := base
		tc.Invocation = invocation
		if result := New().Run(context.Background(), input, tc); !result.IsError || service.calls != 0 {
			t.Fatalf("missing invocation accepted: %+v", result)
		}
	}
	contexts := []tools.Context{base, base, base, base, base}
	contexts[1].Invocation.RunID = "run-2"
	contexts[2].Invocation.ToolUseID = "tool-2"
	contexts[3].Invocation.BatchID = "batch-2"
	contexts[4].SessionID++
	ids := map[string]bool{}
	for _, tc := range contexts {
		result := New().Run(context.Background(), input, tc)
		if result.IsError || service.lastAction.ID == "" || ids[service.lastAction.ID] {
			t.Fatalf("invalid identity: %+v %+v", result, service.lastAction)
		}
		ids[service.lastAction.ID] = true
		if strings.Contains(result.Content, "password-123") {
			t.Fatal("leaked input")
		}
		id := service.lastAction.ID
		New().Run(context.Background(), input, tc)
		if service.lastAction.ID != id {
			t.Fatal("same invocation changed ID")
		}
		if service.lastAction.Text != "password-123" {
			t.Fatal("lost input")
		}
	}
}
func TestRejectsFakeCorruptAndWrongTypePNG(t *testing.T) {
	valid := realPNG(t)
	for _, test := range []struct {
		name      string
		data      []byte
		mediaType string
	}{
		{"fake", []byte("png"), pngMediaType}, {"signature", []byte("\x89PNG\r\n\x1a\n"), pngMediaType},
		{"truncated body", valid[:len(valid)/2], pngMediaType}, {"empty", nil, pngMediaType},
		{"wrong MIME", valid, "image/jpeg"}, {"oversized", make([]byte, maxPNGBytes+1), pngMediaType},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := screenshotService(t)
			service.image = test.data
			service.imageType = test.mediaType
			result := runRequest(service, `{"session_id":"computer-1","action":"observe"}`)
			if !result.IsError || len(result.ContextMessages) != 0 {
				t.Fatalf("bad image accepted: %+v", result)
			}
		})
	}
}
func TestExecutePreservesReceiptAndAfterImageEvenOnError(t *testing.T) {
	for _, outcome := range []cu.Outcome{cu.OutcomeExecuted, cu.OutcomeUnknown} {
		t.Run(string(outcome), func(t *testing.T) {
			service := screenshotService(t)
			service.receipt = cu.ActionReceipt{Outcome: outcome, AfterObservationID: "after-1", BeforeObservationID: "before-1", ErrorMessage: "password-123", RedactedActionSummary: "password-123"}
			if outcome == cu.OutcomeUnknown {
				service.serviceErr = errors.New("password-123")
			}
			result := runRequest(service, `{"session_id":"computer-1","action":"type","text":"password-123"}`)
			if result.IsError != (outcome == cu.OutcomeUnknown) || strings.Contains(result.Content, "password-123") {
				t.Fatalf("wrong error: %+v", result)
			}
			var payload struct {
				Receipt cu.ActionReceipt `json:"receipt"`
			}
			if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Receipt.Outcome != outcome || payload.Receipt.BeforeObservationID != "before-1" || payload.Receipt.AfterObservationID != "after-1" || payload.Receipt.ActionID != service.lastAction.ID {
				t.Fatalf("lost receipt: %+v", payload)
			}
			if service.imageID != "after-1" {
				t.Fatal("wrong after image")
			}
			assertImage(t, result, service.image)
		})
	}
}
func TestServiceErrorsNeverExposeRawStrings(t *testing.T) {
	for _, action := range []string{"observe", "type", "pause", "stop"} {
		t.Run(action, func(t *testing.T) {
			service := screenshotService(t)
			service.serviceErr = errors.New("secret-input")
			result := runRequest(service, `{"session_id":"computer-1","action":"`+action+`"}`)
			if !result.IsError || strings.Contains(result.Content, "secret-input") {
				t.Fatalf("unsafe error: %+v", result)
			}
		})
	}
	service := screenshotService(t)
	service.imageErr = errors.New("secret-input")
	result := runRequest(service, `{"session_id":"computer-1","action":"observe"}`)
	if !result.IsError || strings.Contains(result.Content, "secret-input") {
		t.Fatalf("unsafe image error: %+v", result)
	}
}
func TestAfterImageFailureDoesNotDiscardUnknownReceipt(t *testing.T) {
	service := screenshotService(t)
	service.receipt = cu.ActionReceipt{Outcome: cu.OutcomeUnknown, AfterObservationID: "after-1"}
	service.serviceErr = errors.New("input-secret")
	service.imageErr = errors.New("image-secret")
	result := runRequest(service, `{"session_id":"computer-1","action":"click"}`)
	if !result.IsError || !strings.Contains(result.Content, string(cu.OutcomeUnknown)) || strings.Contains(result.Content, "secret") || len(result.ContextMessages) != 0 {
		t.Fatalf("unsafe receipt: %+v", result)
	}
}
func TestToolIsSerial(t *testing.T) {
	if got := New().ExecutionPolicy().Concurrency; got != tools.ConcurrencySerial {
		t.Fatalf("concurrency = %s", got)
	}
}

// A receipt's after-image is evidence, not a fresh Controller observation:
// BeginAction has already consumed the previous observation. Labeling both
// images alike encouraged live models to skip Observe and get rejected.
func TestScreenshotContextDistinguishesObservationFromActionEvidence(t *testing.T) {
	for _, action := range []string{"observe", "click"} {
		t.Run(action, func(t *testing.T) {
			service := screenshotService(t)
			service.receipt = cu.ActionReceipt{Outcome: cu.OutcomeExecuted, AfterObservationID: "after-1"}
			result := runRequest(service, `{"session_id":"computer-1","action":"`+action+`"}`)
			if result.IsError {
				t.Fatal(result.Content)
			}
			assertImage(t, result, service.image)
			caption := result.ContextMessages[0].Content[0].Text
			for _, want := range []string{"10x10 pixels", "top-left", "Do not divide by scale_factor"} {
				if !strings.Contains(caption, want) {
					t.Errorf("caption missing %q: %s", want, caption)
				}
			}
			if action == "observe" {
				if !strings.Contains(caption, `"obs-1"`) || strings.Contains(caption, "evidence only") {
					t.Errorf("wrong observation caption: %s", caption)
				}
			} else {
				for _, want := range []string{`"after-1"`, "evidence only", "call observe before the next input", "not an actionable observation"} {
					if !strings.Contains(caption, want) {
						t.Errorf("caption missing %q: %s", want, caption)
					}
				}
			}
		})
	}
}
