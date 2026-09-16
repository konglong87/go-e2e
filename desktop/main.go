package main

import (
	"context"
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
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	serverToken = "test-token"
)

//go:embed all:frontend/dist
var bundledAssets embed.FS

type app struct {
	mu     sync.Mutex
	server *exec.Cmd
	config desktopConfig
	port   int
}

func main() {
	port, err := selectServerPort()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	application := &app{port: port}
	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := wails.Run(&options.App{
		Title:      "golang-cc",
		Width:      1440,
		Height:     900,
		MinWidth:   1024,
		MinHeight:  700,
		OnStartup:  application.startup,
		OnShutdown: application.shutdown,
		AssetServer: &assetserver.Options{
			Assets:  bundledAssets,
			Handler: newDesktopHandler(bundledAssets, httputil.NewSingleHostReverseProxy(target)),
		},
		Bind: []interface{}{application},
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (a *app) startup(ctx context.Context) {
	config, err := loadDesktopConfig()
	if err != nil {
		wailsruntime.LogErrorf(ctx, "load desktop config: %v", err)
		return
	}
	if config.Workspace == "" {
		config.Workspace, err = chooseWorkspace(ctx)
		if err != nil {
			wailsruntime.LogErrorf(ctx, "choose workspace: %v", err)
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
		wailsruntime.LogErrorf(ctx, "locate golang-cc executable: %v", err)
		return
	}
	cmd := exec.CommandContext(ctx, executable,
		"--cwd", config.Workspace,
		"server",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(a.port),
		"--auth-token", serverToken,
	)
	sqlitePath, sqliteErr := desktopSQLitePath()
	if sqliteErr != nil {
		wailsruntime.LogErrorf(ctx, "resolve desktop sqlite path: %v", sqliteErr)
		return
	}
	cmd.Env = append(os.Environ(),
		"GOLANG_CC_SQLITE_PATH="+sqlitePath,
		"GOLANG_CC_TENANT_KEY=webui-local",
		"GOLANG_CC_USER_ID=webui-local-user",
	)
	cmd.Dir, _ = os.Getwd()
	if err := cmd.Start(); err != nil {
		wailsruntime.LogErrorf(ctx, "start local golang-cc server: %v", err)
		return
	}
	a.mu.Lock()
	a.server = cmd
	a.mu.Unlock()

	if !waitForServer(ctx, a.port) {
		wailsruntime.LogErrorf(ctx, "local golang-cc server did not become ready on port %d", a.port)
	}
}

func desktopSQLitePath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "golang-cc")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "desktop.sqlite"), nil
}

func (a *app) shutdown(ctx context.Context) {
	a.mu.Lock()
	cmd := a.server
	a.server = nil
	a.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		wailsruntime.LogErrorf(ctx, "stop local golang-cc server: %v", err)
	}
	_ = waitWithTimeout(cmd, 5*time.Second)
}

func locateServerExecutable() (string, error) {
	if path := os.Getenv("GOLANG_CC_SERVER_BINARY"); path != "" {
		return path, nil
	}
	if path, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(path), "golang-cc")
		if stdruntime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("set GOLANG_CC_SERVER_BINARY or place golang-cc beside the desktop binary")
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

func waitForServer(ctx context.Context, port int) bool {
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
			reqCtx, cancel := context.WithTimeout(ctx, time.Second)
			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", port), nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+serverToken)
				resp, requestErr := http.DefaultClient.Do(req)
				if requestErr == nil {
					_ = resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						cancel()
						return true
					}
				}
			}
			cancel()
		}
	}
}

func waitWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return cmd.Process.Kill()
	}
}
