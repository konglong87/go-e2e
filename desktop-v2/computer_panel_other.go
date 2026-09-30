//go:build !darwin

package main

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

func newNativeComputerPanel() computerPanel {
	return nil
}

func (nativeComputerPanel) Update(ComputerPanelSnapshot) {}
func (nativeComputerPanel) Poll() *computerPanelCommand  { return nil }
func (nativeComputerPanel) Close()                       {}
