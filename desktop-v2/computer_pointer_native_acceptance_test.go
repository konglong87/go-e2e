//go:build darwin && computeracceptance

package main

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

// TestNativeComputerPointerScreenshot is an opt-in visual gate. It exercises
// the real AppKit pointer window and saves one native desktop screenshot; it
// does not post input or modify the target application.
func TestNativeComputerPointerScreenshot(t *testing.T) {
	output := os.Getenv("COMPUTER_POINTER_ACCEPTANCE_OUTPUT")
	if output == "" {
		t.Skip("COMPUTER_POINTER_ACCEPTANCE_OUTPUT is not set")
	}
	if !filepath.IsAbs(output) {
		t.Fatal("pointer acceptance output must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		t.Fatal(err)
	}
	pointer := newNativeComputerPointer()
	if pointer == nil {
		t.Fatal("native pointer overlay is unavailable")
	}
	defer pointer.Close()

	now := time.Now()
	point := cu.ScreenPoint{X: 370, Y: 560}
	pointer.Show(cu.ActionVisualCue{
		SessionID: "pointer-acceptance-session", TurnID: "pointer-acceptance-turn",
		ActionID: "pointer-acceptance-action", ActionKind: cu.ActionClick,
		Phase: cu.VisualCueDispatched, DispatchState: cu.DispatchComplete,
		AppBundleID: "com.apple.calculator", WindowID: "acceptance-window",
		GlobalPoint: &point, CreatedAt: now, ExpiresAt: now.Add(2 * time.Second),
	})
	time.Sleep(400 * time.Millisecond)
	if err := exec.Command("screencapture", "-x", output).Run(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("native pointer screenshot missing: %v", err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(file)
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	// The marker is drawn in points while screencapture returns Retina pixels.
	// A locked/permission-denied desktop can still produce a valid all-black
	// PNG, so require visible marker pixels around the expected point.
	visible := false
	centerX, centerY := int(point.X*2), int(point.Y*2)
	for y := centerY - 80; y <= centerY+80 && !visible; y++ {
		for x := centerX - 80; x <= centerX+180; x++ {
			if x < 0 || y < 0 || x >= image.Bounds().Dx() || y >= image.Bounds().Dy() {
				continue
			}
			r, g, b, _ := image.At(x, y).RGBA()
			if r > 18000 || g > 18000 || b > 18000 {
				visible = true
				break
			}
		}
	}
	if !visible {
		t.Fatal("native pointer marker was not visible in the real desktop screenshot")
	}
	pointer.Hide()
}
