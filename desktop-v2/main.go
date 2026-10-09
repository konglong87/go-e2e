package main

import (
	"context"
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/konglong87/go-e2e/internal/computerbridge"
	"github.com/konglong87/go-e2e/internal/sessioncontrol"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macoptions "github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

type app struct {
	mu                sync.Mutex
	config            desktopConfig
	port              int
	token             string
	service           *localServiceController
	computerManager   *computerManager
	acceptanceClose   func()
	computerBridge    *computerbridge.Listener
	windowCtx         context.Context
	windowDone        chan struct{}
	computerPanel     computerPanel
	computerPanelDone chan struct{}
	computerOverlay   computerOverlayState
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
	application := &app{port: port, token: token}
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
	a.startComputerPanelIfEnabled(ctx)
	closeAcceptance, acceptanceErr := startComputerAcceptance(ctx, a.computer(), os.Args[1:])
	if acceptanceErr != nil {
		startupLog("computer acceptance: " + acceptanceErr.Error())
	} else {
		a.acceptanceClose = closeAcceptance
	}
	startupLog("startup begin")
	config, err := loadDesktopConfig()
	if err != nil {
		startupLog("load config: " + err.Error())
		wailsruntime.LogErrorf(ctx, "load desktop config: %v", err)
		return
	}
	restoreWindowGeometry(ctx, config)
	startupLog("config workspace=" + config.Workspace)
	if config.Workspace == "" {
		config.Workspace, err = ensureDefaultWorkspace()
		if err != nil {
			startupLog("create default workspace: " + err.Error())
			wailsruntime.LogErrorf(ctx, "create default workspace: %v", err)
			return
		}
		if err := saveDesktopConfig(config); err != nil {
			wailsruntime.LogErrorf(ctx, "save desktop config: %v", err)
			return
		}
	}
	a.config = config

	executable, err := locateServerExecutable()
	if err != nil {
		startupLog("locate server: " + err.Error())
		wailsruntime.LogErrorf(ctx, "locate go-e2e executable: %v", err)
		return
	}
	startupLog("server executable=" + executable)
	sqlitePath, sqliteErr := desktopSQLitePath()
	if sqliteErr != nil {
		startupLog("sqlite path: " + sqliteErr.Error())
		wailsruntime.LogErrorf(ctx, "resolve desktop sqlite path: %v", sqliteErr)
		return
	}
	startupLog("sqlite path=" + sqlitePath)
	serverLogPath := filepath.Join(filepath.Dir(sqlitePath), "go-e2e-server.log")
	var bridgeConfig *computerbridge.Config
	if stdruntime.GOOS == "darwin" {
		bridge, err := startDesktopComputerBridge(ctx, filepath.Dir(sqlitePath), a.computer())
		if err != nil {
			startupLog("computer bridge unavailable: " + err.Error())
		} else {
			a.mu.Lock()
			a.computerBridge = bridge
			a.mu.Unlock()
			cfg := bridge.Config()
			bridgeConfig = &cfg
		}
	}
	service := newLocalServiceController(localServiceConfig{
		executable:     executable,
		computerBridge: bridgeConfig,
		workspace:      config.Workspace,
		port:           a.port,
		token:          a.token,
		env: []string{
			"GO_E2E_SQLITE_PATH=" + sqlitePath,
			"GOLANG_CC_TENANT_KEY=webui-local",
			"GOLANG_CC_USER_ID=webui-local-user",
			"GOLANG_CC_DESKTOP_MODE=1",
			desktopSessionBackendEnv + "=" + desktopSessionBackend(config),
			// Desktop MVP does not expose scheduled jobs. Disabling the scheduler
			// keeps startup independent from stale daemon locks left by a crashed
			// desktop/server process.
			"GOLANG_CC_DISABLE_SCHEDULER=1",
		},
		dir:     func() string { value, _ := os.Getwd(); return value }(),
		logPath: serverLogPath,
	})
	a.mu.Lock()
	a.service = service
	a.mu.Unlock()
	if err := service.start(); err != nil {
		startupLog("start local service: " + err.Error())
		wailsruntime.LogErrorf(ctx, "start local go-e2e server: %v", err)
	}
}

func desktopSessionBackend(config desktopConfig) string {
	if value := strings.TrimSpace(os.Getenv(desktopSessionBackendEnv)); value != "" {
		return value
	}
	if value := strings.TrimSpace(config.SessionBackend); value != "" {
		return value
	}
	return defaultSessionBackend
}

func (a *app) GetSessionBackend() string {
	a.mu.Lock()
	config := a.config
	a.mu.Unlock()
	return desktopSessionBackend(config)
}

func (a *app) SetSessionBackend(value string) error {
	backend, err := sessioncontrol.ParseSessionBackend(value)
	if err != nil {
		return err
	}
	config, err := loadDesktopConfig()
	if err != nil {
		return fmt.Errorf("load desktop config: %w", err)
	}
	config.SessionBackend = backend.String()
	if err := saveDesktopConfig(config); err != nil {
		return fmt.Errorf("save desktop session backend: %w", err)
	}

	a.mu.Lock()
	a.config = config
	service := a.service
	a.mu.Unlock()
	if service == nil {
		return nil
	}
	service.setSessionBackend(backend.String())
	if err := a.RestartLocalService(); err != nil {
		return fmt.Errorf("restart local service after session backend change: %w", err)
	}
	return nil
}

func (a *app) domReady(ctx context.Context) {
	a.setWindowContext(ctx)
	script := fmt.Sprintf(`window.__GO_E2E_DESKTOP_TOKEN__=%q; window.dispatchEvent(new Event("go-e2e-desktop-token"));`, a.token)
	wailsruntime.WindowExecJS(ctx, script)
	a.startWindowStateWatcher(ctx)
}

func (a *app) SelectWorkspace() (string, error) {
	ctx := a.windowContext()
	if ctx == nil {
		return "", errors.New("desktop window is not ready")
	}
	return selectWorkspace(ctx)
}

func (a *app) beforeClose(ctx context.Context) bool {
	startupLog("before close")
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
	startupLog("shutdown begin")
	defer startupLog("shutdown complete")
	if a.acceptanceClose != nil {
		a.acceptanceClose()
	}
	if err := a.persistWindowState(ctx); err != nil {
		wailsruntime.LogErrorf(ctx, "save window state: %v", err)
	}
	a.stopWindowStateWatcher()
	a.stopComputerPanel()
	a.mu.Lock()
	service := a.service
	computer := a.computerManager
	bridge := a.computerBridge
	a.computerBridge = nil
	a.service = nil
	a.computerManager = nil
	a.mu.Unlock()
	if bridge != nil {
		_ = bridge.Close()
	}
	if computer != nil {
		if err := computer.close(ctx); err != nil {
			wailsruntime.LogErrorf(ctx, "stop computer helper: %v", err)
		}
	}
	if service != nil {
		if err := service.stop(ctx); err != nil {
			wailsruntime.LogErrorf(ctx, "stop local go-e2e server: %v", err)
		}
	}
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
