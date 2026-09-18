package main

import (
	"context"
	"crypto/rand"
	"embed"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macoptions "github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

type app struct {
	mu          sync.Mutex
	server      *exec.Cmd
	done        chan error
	config      desktopConfig
	port        int
	token       string
	windowCtx   context.Context
	windowDone  chan struct{}
	serverReady chan struct{}
	readyOnce   sync.Once
}

func main() {
	port, err := selectServerPort()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	token, err := newServerToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	application := &app{port: port, token: token, serverReady: make(chan struct{})}
	config, err := loadDesktopConfig()
	if err != nil {
		startupLog("load desktop config for window: " + err.Error())
		config = desktopConfig{}
	}
	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := wails.Run(desktopWailsOptions(application, config, target)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func desktopWailsOptions(application *app, config desktopConfig, target *url.URL) *options.App {
	return &options.App{
		Title:            "go-e2e",
		Width:            windowWidth(config),
		Height:           windowHeight(config),
		MinWidth:         minWindowWidth,
		MinHeight:        minWindowHeight,
		WindowStartState: windowStartState(config),
		// Wails disables macOS's native Zoom button when Mac options are nil.
		// Keep it enabled so the green button exposes the system fullscreen menu.
		Mac:           &macoptions.Options{DisableZoom: false},
		OnStartup:     application.startup,
		OnDomReady:    application.domReady,
		OnBeforeClose: application.beforeClose,
		OnShutdown:    application.shutdown,
		AssetServer: &assetserver.Options{
			Assets:  bundledAssets,
			Handler: httputil.NewSingleHostReverseProxy(target),
		},
		Bind: []interface{}{application},
	}
}

func (a *app) startup(ctx context.Context) {
	a.setWindowContext(ctx)
	startupLog("startup begin")
	config, err := loadDesktopConfig()
	if err != nil {
		startupLog("load config: " + err.Error())
		wailsruntime.LogErrorf(ctx, "load desktop config: %v", err)
		a.signalServerReady()
		return
	}
	restoreWindowGeometry(ctx, config)
	startupLog("config workspace=" + config.Workspace)
	if config.Workspace == "" {
		config.Workspace, err = chooseWorkspace(ctx)
		if err != nil {
			startupLog("choose workspace: " + err.Error())
			wailsruntime.LogErrorf(ctx, "choose workspace: %v", err)
			a.signalServerReady()
			return
		}
		if err := saveDesktopConfig(config); err != nil {
			wailsruntime.LogErrorf(ctx, "save desktop config: %v", err)
			a.signalServerReady()
			return
		}
	}
	a.config = config

	executable, err := locateServerExecutable()
	if err != nil {
		startupLog("locate server: " + err.Error())
		wailsruntime.LogErrorf(ctx, "locate go-e2e executable: %v", err)
		a.signalServerReady()
		return
	}
	startupLog("server executable=" + executable)
	// Wails' startup context is scoped to the startup callback. The server
	// must outlive that callback and be stopped explicitly from OnShutdown.
	cmd := exec.Command(executable,
		"--cwd", config.Workspace,
		"server",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(a.port),
		"--auth-token", a.token,
	)
	sqlitePath, sqliteErr := desktopSQLitePath()
	if sqliteErr != nil {
		startupLog("sqlite path: " + sqliteErr.Error())
		wailsruntime.LogErrorf(ctx, "resolve desktop sqlite path: %v", sqliteErr)
		a.signalServerReady()
		return
	}
	startupLog("sqlite path=" + sqlitePath)
	cmd.Env = append(os.Environ(),
		"GOLANG_CC_SQLITE_PATH="+sqlitePath,
		"GOLANG_CC_TENANT_KEY=webui-local",
		"GOLANG_CC_USER_ID=webui-local-user",
		// Desktop MVP does not expose scheduled jobs. Disabling the scheduler
		// keeps startup independent from stale daemon locks left by a crashed
		// desktop/server process.
		"GOLANG_CC_DISABLE_SCHEDULER=1",
	)
	serverLogPath := filepath.Join(filepath.Dir(sqlitePath), "go-e2e-server.log")
	serverLog, logErr := os.OpenFile(serverLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if logErr != nil {
		startupLog("open server log: " + logErr.Error())
		wailsruntime.LogErrorf(ctx, "open server log: %v", logErr)
		a.signalServerReady()
		return
	}
	defer func() { _ = serverLog.Close() }()
	cmd.Stdout = serverLog
	cmd.Stderr = serverLog
	cmd.Dir, _ = os.Getwd()
	if err := cmd.Start(); err != nil {
		startupLog("start server: " + err.Error())
		wailsruntime.LogErrorf(ctx, "start local go-e2e server: %v", err)
		a.signalServerReady()
		return
	}
	startupLog(fmt.Sprintf("server started pid=%d port=%d", cmd.Process.Pid, a.port))
	a.mu.Lock()
	a.server = cmd
	a.done = make(chan error, 1)
	done := a.done
	a.mu.Unlock()
	go func() {
		err := cmd.Wait()
		done <- err
		startupLog(fmt.Sprintf("server exited pid=%d err=%v", cmd.Process.Pid, err))
	}()

	if !a.waitForServer(ctx) {
		readyErr := fmt.Errorf("local go-e2e server did not become ready on port %d", a.port)
		if state := cmd.ProcessState; state != nil && state.Exited() {
			wailsruntime.LogErrorf(ctx, "local go-e2e server exited before readiness with code %d", state.ExitCode())
			readyErr = fmt.Errorf("local go-e2e server exited before readiness with code %d", state.ExitCode())
		}
		wailsruntime.LogErrorf(ctx, "%v", readyErr)
		startupLog(fmt.Sprintf("server not ready port=%d", a.port))
		a.signalServerReady()
	} else {
		startupLog(fmt.Sprintf("server ready port=%d", a.port))
		a.signalServerReady()
	}
}

func (a *app) domReady(ctx context.Context) {
	a.setWindowContext(ctx)
	go func() {
		if a.serverReady != nil {
			select {
			case <-a.serverReady:
			case <-ctx.Done():
				return
			}
		}
		script := fmt.Sprintf(`window.__GO_E2E_DESKTOP_TOKEN__=%q; window.dispatchEvent(new Event("go-e2e-desktop-token"));`, a.token)
		wailsruntime.WindowExecJS(ctx, script)
	}()
	a.startWindowStateWatcher(ctx)
}

func (a *app) signalServerReady() {
	if a.serverReady == nil {
		return
	}
	a.readyOnce.Do(func() {
		close(a.serverReady)
	})
}

func (a *app) beforeClose(ctx context.Context) bool {
	if err := a.persistWindowState(ctx); err != nil {
		wailsruntime.LogErrorf(ctx, "save window state: %v", err)
	}
	return false
}

func newServerToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate desktop auth token: %w", err)
	}
	return fmt.Sprintf("%x", raw), nil
}

func startupLog(message string) {
	dir, err := desktopDataDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "go-e2e-startup.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s %s\n", time.Now().Format(time.RFC3339Nano), message)
}

func desktopSQLitePath() (string, error) {
	dir, err := desktopDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "go-e2e.sqlite"), nil
}

func (a *app) shutdown(ctx context.Context) {
	if err := a.persistWindowState(ctx); err != nil {
		wailsruntime.LogErrorf(ctx, "save window state: %v", err)
	}
	a.stopWindowStateWatcher()
	a.mu.Lock()
	cmd := a.server
	done := a.done
	a.server = nil
	a.done = nil
	a.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		wailsruntime.LogErrorf(ctx, "stop local go-e2e server: %v", err)
	}
	_ = waitWithTimeout(done, cmd, 5*time.Second)
}

func locateServerExecutable() (string, error) {
	for _, envName := range []string{"GO_E2E_SERVER_BINARY", "GOLANG_CC_SERVER_BINARY"} {
		if path := os.Getenv(envName); path != "" {
			return path, nil
		}
	}
	if path, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(path), "go-e2e")
		if stdruntime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("set GO_E2E_SERVER_BINARY or place go-e2e beside the desktop binary")
}

func serverPort() int {
	if value, err := strconv.Atoi(os.Getenv("GOLANG_CC_DESKTOP_SERVER_PORT")); err == nil && value > 0 && value < 65536 {
		return value
	}
	return 0
}

func selectServerPort() (int, error) {
	if port := serverPort(); port != 0 {
		return port, nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("select desktop server port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("release desktop server port: %w", err)
	}
	return port, nil
}

func (a *app) waitForServer(ctx context.Context) bool {
	deadline := time.NewTimer(15 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case <-ticker.C:
			if a.serverEndpointReady(ctx, "/health") && a.serverEndpointReady(ctx, "/readyz") {
				return true
			}
		}
	}
}

func (a *app) serverEndpointReady(ctx context.Context, path string) bool {
	reqCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", a.port, path), nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitWithTimeout(done <-chan error, cmd *exec.Cmd, timeout time.Duration) error {
	if done == nil {
		return nil
	}
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return cmd.Process.Kill()
	}
}
