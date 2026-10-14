package main

import (
	"context"
	"sync"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

// computerPointer is deliberately narrower than the Computer Use authority.
// It can only render/hide a visual cue and never posts input or changes a turn.
type computerPointer interface {
	Show(cu.ActionVisualCue)
	Hide()
	Close()
}

func startComputerVisualCueOverlay(ctx context.Context, bus *cu.VisualCueBus) func() {
	return startComputerVisualCueOverlayWithPointer(ctx, bus, newNativeComputerPointer())
}

func startComputerVisualCueOverlayWithPointer(ctx context.Context, bus *cu.VisualCueBus, pointer computerPointer) func() {
	if ctx == nil {
		ctx = context.Background()
	}
	if bus == nil || pointer == nil {
		return func() {}
	}
	cues, cancel := bus.SubscribeVisualCues(8)
	stop := make(chan struct{})
	done := make(chan struct{})
	var stopOnce sync.Once
	go func() {
		defer close(done)
		defer cancel()
		defer pointer.Hide()
		defer pointer.Close()

		var timer *time.Timer
		var timerC <-chan time.Time
		resetTimer := func(deadline time.Time) {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				pointer.Hide()
				timerC = nil
				return
			}
			if timer == nil {
				timer = time.NewTimer(remaining)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(remaining)
			}
			timerC = timer.C
		}
		stopTimer := func() {
			if timer == nil {
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timerC = nil
		}
		defer stopTimer()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-timerC:
				pointer.Hide()
				timerC = nil
			case cue, ok := <-cues:
				if !ok {
					return
				}
				if cue.GlobalPoint == nil || !cue.ExpiresAt.After(time.Now()) {
					continue
				}
				pointer.Show(cue)
				resetTimer(cue.ExpiresAt)
			}
		}
	}()
	return func() {
		stopOnce.Do(func() { close(stop) })
		<-done
	}
}
