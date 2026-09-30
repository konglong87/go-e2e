package main

import (
	"context"
	"strings"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

const computerPanelPollInterval = 60 * time.Millisecond

func (a *app) startComputerPanel(ctx context.Context) {
	panel := newNativeComputerPanel()
	if panel == nil {
		return
	}
	done := make(chan struct{})
	a.mu.Lock()
	if a.computerPanel != nil {
		a.mu.Unlock()
		panel.Close()
		return
	}
	a.computerPanel = panel
	a.computerPanelDone = done
	a.mu.Unlock()

	go func() {
		ticker := time.NewTicker(computerPanelPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				a.handleComputerPanelCommand(ctx)
			}
		}
	}()
}

func (a *app) stopComputerPanel() {
	a.mu.Lock()
	panel, done := a.computerPanel, a.computerPanelDone
	a.computerPanel, a.computerPanelDone = nil, nil
	a.mu.Unlock()
	if done != nil {
		close(done)
	}
	if panel != nil {
		panel.Close()
	}
}

// UpdateComputerOverlay is a display-only bridge. It never creates a session,
// captures a screenshot, or changes Computer Use authority.
func (a *app) UpdateComputerOverlay(snapshot ComputerPanelSnapshot) {
	a.mu.Lock()
	panel := a.computerPanel
	a.mu.Unlock()
	if panel != nil {
		panel.Update(snapshot)
	}
}

func (a *app) handleComputerPanelCommand(ctx context.Context) {
	a.mu.Lock()
	panel := a.computerPanel
	a.mu.Unlock()
	if panel == nil {
		return
	}
	command := panel.Poll()
	if command == nil {
		return
	}
	switch command.Kind {
	case computerPanelCommandDismiss:
		panel.Update(ComputerPanelSnapshot{Visible: false})
	case computerPanelCommandStop, computerPanelCommandPause, computerPanelCommandResume:
		if strings.TrimSpace(command.SessionID) == "" {
			return
		}
		kind, ok := mapComputerPanelCommand(command.Kind)
		if !ok {
			return
		}
		_, _ = a.computer().control(ctx, command.SessionID, kind)
	}
}

func mapComputerPanelCommand(kind string) (cu.ActionKind, bool) {
	switch kind {
	case computerPanelCommandStop:
		return cu.ActionStop, true
	case computerPanelCommandPause:
		return cu.ActionPause, true
	case computerPanelCommandResume:
		return cu.ActionResume, true
	default:
		return "", false
	}
}
