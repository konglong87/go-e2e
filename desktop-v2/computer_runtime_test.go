package main

import (
	"context"
	"io"
	"net/http"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
	"testing"
)

func TestComputerRuntimeOutlivesWindowContext(t *testing.T) {
	windowCtx, cancelWindow := context.WithCancel(context.Background())
	runtime := newComputerRuntime(context.Background())
	defer func() { _ = runtime.Stop(context.Background()) }()

	cancelWindow()
	select {
	case <-runtime.Context().Done():
		t.Fatal("computer runtime was coupled to the window context")
	default:
	}
	if windowCtx.Err() == nil {
		t.Fatal("window context did not cancel")
	}
	if runtime.Manager() == nil || runtime.Manager().runtimeContext() != runtime.Context() {
		t.Fatal("manager is not bound to computerRuntimeCtx")
	}
}

func TestComputerHelperLifetimeDoesNotComeFromUIClient(t *testing.T) {
	for _, surface := range []string{"capabilities-first", "conversation-approval"} {
		t.Run(surface, func(t *testing.T) {
			windowCtx, cancelWindow := context.WithCancel(context.Background())
			defer cancelWindow()
			runtime := newComputerRuntime(nil)
			defer func() { _ = runtime.Stop(context.Background()) }()
			m, _ := testComputerManager()
			m.runtimeCtx = runtime.Context()
			runtime.manager = m
			var helperLifetime context.Context
			factory := m.factory
			m.factory = func(ctx context.Context) (cu.Backend, error) {
				helperLifetime = ctx
				return factory(ctx)
			}
			a := &app{windowCtx: windowCtx, computerRuntime: runtime, computerManager: m, token: "synthetic-test-only"}
			if surface == "capabilities-first" {
				if capabilities := a.GetComputerCapabilities(); !capabilities.Available {
					t.Fatal("UI capability probe failed")
				}
			} else {
				a.port = computerOwnerTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"data":{"tenant_id":7,"user_id":11,"session_id":93}}`)
				})
				if _, err := a.StartComputerSession(ComputerSessionStartInput{Approved: true, ConversationRef: "tenant:alpha"}); err != nil {
					t.Fatal(err)
				}
			}
			cancelWindow()
			if helperLifetime == nil || helperLifetime != runtime.Context() || helperLifetime.Err() != nil {
				t.Fatal("UI client cancellation owns the long-lived helper context")
			}
			if err := runtime.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			if helperLifetime.Err() == nil {
				t.Fatal("explicit runtime Stop did not end helper lifetime")
			}
		})
	}
}

type runtimeDrainBackend struct {
	*cu.FakeBackend
	lifetime         context.Context
	closedWhileAlive bool
}

func (b *runtimeDrainBackend) Close(ctx context.Context) error {
	b.closedWhileAlive = b.lifetime.Err() == nil && ctx.Err() == nil
	return b.FakeBackend.Close(ctx)
}

func TestComputerRuntimeStopDrainsBeforeCancellingHelperLifetime(t *testing.T) {
	runtime := newComputerRuntime(nil)
	defer func() { _ = runtime.Stop(context.Background()) }()
	m, _ := testComputerManager()
	m.runtimeCtx = runtime.Context()
	runtime.manager = m
	factory := m.factory
	var backend *runtimeDrainBackend
	m.factory = func(ctx context.Context) (cu.Backend, error) {
		base, err := factory(ctx)
		if err != nil {
			return nil, err
		}
		backend = &runtimeDrainBackend{FakeBackend: base.(*cu.FakeBackend), lifetime: ctx}
		return backend, nil
	}
	if _, err := m.start(context.Background(), ComputerSessionStartInput{Approved: true}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !backend.closedWhileAlive || runtime.Context().Err() == nil {
		t.Fatal("runtime canceled helper before bounded Stop/drain completed")
	}
}

func TestActiveComputerRuntimePreventsWailsHostClose(t *testing.T) {
	manager, _ := testComputerManager()
	ready, err := manager.start(context.Background(), ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	app := &app{computerManager: manager}
	if !app.shouldPreventWindowClose() {
		t.Fatal("active computer session allowed Wails host close")
	}
	if _, err := manager.control(context.Background(), ready.ID, cu.ActionStop); err != nil {
		t.Fatal(err)
	}
	if app.shouldPreventWindowClose() {
		t.Fatal("terminal computer session still blocked explicit host close")
	}
}
