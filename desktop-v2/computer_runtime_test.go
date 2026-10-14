package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
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

func TestProviderRuntimeActiveSessionPreventsWailsHostClose(t *testing.T) {
	manager, _ := testComputerManager()
	ready, err := manager.start(context.Background(), ComputerSessionStartInput{Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	provider := newComputerRuntime(nil)
	provider.manager = manager
	app := &app{providerRuntime: provider}
	if !app.shouldPreventWindowClose() {
		t.Fatal("active provider runtime session allowed Wails host close")
	}
	if _, err := manager.control(context.Background(), ready.ID, cu.ActionStop); err != nil {
		t.Fatal(err)
	}
	if app.shouldPreventWindowClose() {
		t.Fatal("stopped provider runtime still blocked explicit host close")
	}
}

func TestParseComputerRuntimeProcessArgsRequiresPrivateAbsoluteDirectory(t *testing.T) {
	if _, found, err := parseComputerRuntimeProcessArgs(nil); found || err != nil {
		t.Fatalf("no runtime args = found=%t err=%v", found, err)
	}
	if _, found, err := parseComputerRuntimeProcessArgs([]string{computerRuntimeProcessFlag}); !found || err == nil {
		t.Fatal("runtime process without data directory was accepted")
	}
	path, found, err := parseComputerRuntimeProcessArgs([]string{computerRuntimeProcessFlag, computerRuntimeDataDirFlag, "/private/runtime"})
	if err != nil || !found || path != "/private/runtime" {
		t.Fatalf("runtime args = path=%q found=%t err=%v", path, found, err)
	}
	if _, found, err := parseComputerRuntimeProcessArgs([]string{computerRuntimeProcessFlag, computerRuntimeDataDirFlag, "relative"}); !found || err == nil {
		t.Fatal("relative runtime directory was accepted")
	}
}

func TestAppendRuntimeProcessEnvRemovesInheritedHandshakeFD(t *testing.T) {
	env := appendRuntimeProcessEnv([]string{"A=1", "GO_E2E_COMPUTER_RUNTIME_CONFIG_FD=99", "B=2"})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "GO_E2E_COMPUTER_RUNTIME_CONFIG_FD=") || !strings.Contains(joined, "A=1") || !strings.Contains(joined, "B=2") {
		t.Fatalf("sanitized runtime environment = %q", joined)
	}
}

func TestStandaloneRuntimeAcceptsExistingDesktopDataParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(parent, "computer-runtime")
	if err := ensurePrivateDirectory(child); err != nil {
		t.Fatalf("private runtime child rejected under existing data parent: %v", err)
	}
	info, err := os.Stat(child)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("runtime child mode = %o, want 0700", info.Mode().Perm())
	}
}
