package computeruse

import (
	"testing"
	"time"
)

func TestTargetBindingMatchesWindowServerIdentity(t *testing.T) {
	window := WindowRef{ID: "window-1", OwnerPID: 42, BundleID: "com.example.app", IsVisible: true, Frame: &WindowFrame{X: 1, Y: 2, Width: 300, Height: 200}}
	bound, err := NewTargetBinding(window, 7, time.Unix(100, 0))
	if err != nil || !bound.Valid() {
		t.Fatalf("binding=%+v err=%v", bound, err)
	}
	if !bound.Matches(window) {
		t.Fatal("same window identity did not match")
	}
	changed := window
	changed.OwnerPID++
	if bound.Matches(changed) {
		t.Fatal("owner PID change matched stale binding")
	}
	changed = window
	changed.Frame = &WindowFrame{X: 1, Y: 2, Width: 301, Height: 200}
	if bound.Matches(changed) {
		t.Fatal("frame change matched stale binding")
	}
}
