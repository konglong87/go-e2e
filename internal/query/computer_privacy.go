package query

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"

	"github.com/konglong87/go-e2e/internal/anthropic"
	"github.com/konglong87/go-e2e/internal/config"
	"github.com/konglong87/go-e2e/internal/hooks"
	"github.com/konglong87/go-e2e/internal/tools"
)

func redactComputerToolCall(block anthropic.ContentBlock) anthropic.ContentBlock {
	if tools.IsComputerUseTool(block.Name) {
		block.Input = tools.RedactToolInput(block.Name, block.Input)
	}
	return block
}
func redactComputerAssistant(message anthropic.MessageParam) anthropic.MessageParam {
	blocks := append([]anthropic.ContentBlock(nil), message.Content...)
	for i, block := range blocks {
		if block.Type == blockTypeToolUse {
			blocks[i] = redactComputerToolCall(block)
		}
	}
	message.Content = blocks
	return message
}

// Runner normally lets one hook's UpdatedInput flow to the next hook. Desktop
// input is different: each hook gets only the same non-executable summary. No
// hook can read raw input, rewrite it, or send arbitrary text to event sinks.
func runComputerHooks(ctx context.Context, runner hooks.Runner, event, cwd string, payload hooks.Payload) (hooks.Result, error) {
	payload.Input = tools.RedactToolInput(payload.ToolName, payload.Input)
	if payload.Result != "" {
		summary, _ := json.Marshal(struct {
			IsError bool `json:"is_error"`
		}{payload.IsError})
		payload.Result = string(summary)
	}
	var result hooks.Result
	for _, command := range runner.Hooks[event] {
		isolated := hooks.New(map[string][]config.HookCommand{event: {command}})
		next, err := isolated.RunWithPayload(ctx, event, cwd, payload)
		if err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, errors.New("ComputerUse hook failed")
		}
		switch strings.ToLower(strings.TrimSpace(next.PermissionDecision)) {
		case "deny":
			// A later hook cannot undo a denial either.
			result.PermissionDecision = "deny"
			result.PermissionDecisionReason = "ComputerUse denied by hook"
		case "allow", "ask":
			if result.PermissionDecision != "deny" {
				result.PermissionDecision = strings.ToLower(strings.TrimSpace(next.PermissionDecision))
			}
		}
	}
	return result, nil
}

// Keep only content hashes, not screenshot bytes. This also covers retained
// screenshots being recorded again by either v1 or v2 compaction paths.
func (s *Session) rememberTransientComputerImages(messages []anthropic.MessageParam) {
	s.toolStateMu.Lock()
	defer s.toolStateMu.Unlock()
	for _, message := range messages {
		for _, block := range message.Content {
			if block.Type != blockTypeImage || block.Source == nil {
				continue
			}
			if s.transientComputerImages == nil {
				s.transientComputerImages = make(map[[sha256.Size]byte]struct{})
			}
			s.transientComputerImages[sha256.Sum256([]byte(block.Source.Data))] = struct{}{}
		}
	}
}
func (s *Session) isTransientComputerImage(block anthropic.ContentBlock) bool {
	if block.Source == nil {
		return false
	}
	s.toolStateMu.Lock()
	defer s.toolStateMu.Unlock()
	_, ok := s.transientComputerImages[sha256.Sum256([]byte(block.Source.Data))]
	return ok
}

// Post-hook failures do not undo a dispatched action or erase its receipt/image.
func toolHookFailure(name string, result tools.Result, err error) tools.Result {
	if !tools.IsComputerUseTool(name) {
		return tools.Result{Content: err.Error(), IsError: true}
	}
	result.IsError = true
	if result.Content == "" {
		result.Content = "ComputerUse hook failed"
	}
	return result
}
