package main

import (
	"context"
	"sync"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

// RuntimeSupervisor owns the long-lived Computer Use runtime. Its context is
// deliberately independent from the Wails window context; UI reload/hide only
// disconnects a client and cannot cancel an active provider/helper session.
type RuntimeSupervisor struct {
	mu                 sync.Mutex
	computerRuntimeCtx context.Context
	cancel             context.CancelFunc
	manager            *computerManager
	stopped            bool
}

func newComputerRuntime(_ context.Context) *RuntimeSupervisor {
	ctx, cancel := context.WithCancel(context.Background())
	bus := cu.NewEventBus()
	return &RuntimeSupervisor{computerRuntimeCtx: ctx, cancel: cancel, manager: newComputerManagerWithRuntime(ctx, bus)}
}

func (r *RuntimeSupervisor) Context() context.Context {
	if r == nil {
		return context.Background()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.computerRuntimeCtx
}

func (r *RuntimeSupervisor) Manager() *computerManager {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.manager
}

func (r *RuntimeSupervisor) EventBus() *cu.EventBus {
	if manager := r.Manager(); manager != nil {
		return manager.events
	}
	return nil
}

// Stop is the only normal runtime termination path. It revokes the active
// session and closes the helper after canceling the runtime context.
func (r *RuntimeSupervisor) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return nil
	}
	r.stopped = true
	cancel := r.cancel
	manager := r.manager
	r.mu.Unlock()
	cancel()
	if manager == nil {
		return nil
	}
	return manager.close(ctx)
}
