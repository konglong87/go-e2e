package main

import (
	"context"
	"sync"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbridge"
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
	process            *computerRuntimeProcess
	bridgeConfig       *computerbridge.Config
	statusClient       *computerbridge.Client
	visualCueBus       *cu.VisualCueBus
	visualCueStop      func()
	stopped            bool
}

func newComputerRuntime(_ context.Context) *RuntimeSupervisor {
	ctx, cancel := context.WithCancel(context.Background())
	bus := cu.NewEventBus()
	manager := newComputerManagerWithRuntime(ctx, bus)
	return &RuntimeSupervisor{computerRuntimeCtx: ctx, cancel: cancel, manager: manager, visualCueBus: manager.visualCues}
}

func newProcessComputerRuntime(process *computerRuntimeProcess) *RuntimeSupervisor {
	ctx, cancel := context.WithCancel(context.Background())
	config := process.Config()
	return &RuntimeSupervisor{
		computerRuntimeCtx: ctx,
		cancel:             cancel,
		process:            process,
		bridgeConfig:       &config,
		statusClient:       process.Client(),
		visualCueBus:       cu.NewVisualCueBus(),
	}
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

// BridgeConfig returns the in-memory authenticated endpoint owned by a
// standalone runtime process. The token is never written to disk or logged.
// BridgeClient is the authenticated client for the standalone provider
// runtime. It is exposed only to the Wails client adapter; callers cannot mint
// or alter its token.
func (r *RuntimeSupervisor) BridgeClient() *computerbridge.Client {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.statusClient
}

func (r *RuntimeSupervisor) BridgeConfig() (computerbridge.Config, bool) {
	if r == nil {
		return computerbridge.Config{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bridgeConfig == nil {
		return computerbridge.Config{}, false
	}
	return *r.bridgeConfig, true
}

// HasActiveSession reports whether this runtime still owns an authorized,
// non-terminal Computer Use session. The Wails client uses it only as a
// shutdown guard; it never grants or changes runtime authority.
func (r *RuntimeSupervisor) HasActiveSession() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	manager, client := r.manager, r.statusClient
	r.mu.Unlock()
	if manager != nil {
		return manager.hasActiveSession()
	}
	if client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, err := client.RuntimeStatus(ctx, computerRuntimeSupervisionOwner)
	return err == nil && status.ActiveSession
}

func (r *RuntimeSupervisor) EventBus() *cu.EventBus {
	if manager := r.Manager(); manager != nil {
		return manager.events
	}
	return nil
}

func (r *RuntimeSupervisor) VisualCueBus() *cu.VisualCueBus {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.visualCueBus != nil {
		return r.visualCueBus
	}
	if r.manager != nil {
		return r.manager.visualCues
	}
	return nil
}

// StartVisualCueOverlay attaches a best-effort native pointer to this runtime.
// It is explicit so unit-created supervisors do not create windows implicitly.
func (r *RuntimeSupervisor) StartVisualCueOverlay() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.visualCueStop != nil {
		r.mu.Unlock()
		return
	}
	ctx, manager, client, bus := r.computerRuntimeCtx, r.manager, r.statusClient, r.visualCueBus
	if bus == nil && manager != nil {
		bus = manager.visualCues
		r.visualCueBus = bus
	}
	r.mu.Unlock()
	if bus == nil {
		return
	}
	var stop func()
	if manager != nil {
		stop = startComputerVisualCueOverlay(ctx, bus)
	} else if client != nil {
		stop = startRemoteComputerVisualCueOverlay(ctx, client, bus)
	} else {
		return
	}
	r.mu.Lock()
	if r.visualCueStop == nil && !r.stopped {
		r.visualCueStop = stop
		stop = nil
	}
	r.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// Stop is the only normal runtime termination path. It revokes the active
// session and drains the helper before canceling its lifetime context. Client
// disconnect cannot interrupt this bounded, explicit shutdown.
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
	process := r.process
	visualCueStop := r.visualCueStop
	r.visualCueStop = nil
	r.mu.Unlock()
	if visualCueStop != nil {
		visualCueStop()
	}
	defer cancel()
	if process != nil {
		return process.Stop(ctx)
	}
	if manager == nil {
		return nil
	}
	stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(ctx), computerRequestTimeout)
	defer stopCancel()
	return manager.close(stopCtx)
}
