//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include "computer_panel_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"sync"
)

const (
	computerPanelCommandStop     = "stop"
	computerPanelCommandPause    = "pause"
	computerPanelCommandResume   = "resume"
	computerPanelCommandExpand   = "expand"
	computerPanelCommandCollapse = "collapse"
	computerPanelCommandDismiss  = "dismiss"
)

type ComputerPanelSnapshot struct {
	Visible   bool   `json:"visible"`
	Expanded  bool   `json:"expanded"`
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	ImageData string `json:"image_data"`
	CanStop   bool   `json:"can_stop"`
	CanPause  bool   `json:"can_pause"`
	CanResume bool   `json:"can_resume"`
	Language  string `json:"language"`
}

type computerPanelCommand struct {
	Kind      string `json:"kind"`
	SessionID string `json:"session_id"`
}

type computerPanel interface {
	Update(ComputerPanelSnapshot)
	Poll() *computerPanelCommand
	Close()
}

type nativeComputerPanel struct{}

type nativeComputerPanelState struct {
	handle *C.goe2e_computer_panel

	mu      sync.Mutex
	closed  bool
	hasLast bool
	last    ComputerPanelSnapshot
}

var nativeComputerPanelStates sync.Map // map[*nativeComputerPanel]*nativeComputerPanelState

func newNativeComputerPanel() computerPanel {
	handle := C.goe2e_computer_panel_create()
	if handle == nil {
		return nil
	}
	panel := &nativeComputerPanel{}
	nativeComputerPanelStates.Store(panel, &nativeComputerPanelState{handle: handle})
	return panel
}

func (p *nativeComputerPanel) state() *nativeComputerPanelState {
	if p == nil {
		return nil
	}
	value, ok := nativeComputerPanelStates.Load(p)
	if !ok {
		return nil
	}
	state, _ := value.(*nativeComputerPanelState)
	return state
}

func (p *nativeComputerPanel) Update(snapshot ComputerPanelSnapshot) {
	state := p.state()
	if state == nil {
		return
	}
	state.mu.Lock()
	if state.closed || (state.hasLast && state.last == snapshot) {
		state.mu.Unlock()
		return
	}
	state.last = snapshot
	state.hasLast = true
	handle := state.handle
	state.mu.Unlock()

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	cPayload := C.CBytes(payload)
	defer C.free(cPayload)
	C.goe2e_computer_panel_update(handle, (*C.char)(cPayload))
}

func (p *nativeComputerPanel) Poll() *computerPanelCommand {
	state := p.state()
	if state == nil {
		return nil
	}
	state.mu.Lock()
	if state.closed {
		state.mu.Unlock()
		return nil
	}
	handle := state.handle
	state.mu.Unlock()

	raw := C.goe2e_computer_panel_poll(handle)
	if raw == nil {
		return nil
	}
	defer C.goe2e_computer_panel_free_string(raw)

	var command computerPanelCommand
	if err := json.Unmarshal([]byte(C.GoString(raw)), &command); err != nil {
		return nil
	}
	switch command.Kind {
	case computerPanelCommandStop, computerPanelCommandPause, computerPanelCommandResume,
		computerPanelCommandExpand, computerPanelCommandCollapse, computerPanelCommandDismiss:
		return &command
	default:
		return nil
	}
}

func (p *nativeComputerPanel) Close() {
	state := p.state()
	if state == nil {
		return
	}
	state.mu.Lock()
	if state.closed {
		state.mu.Unlock()
		return
	}
	state.closed = true
	handle := state.handle
	state.mu.Unlock()

	C.goe2e_computer_panel_close(handle)
	nativeComputerPanelStates.Delete(p)
}
