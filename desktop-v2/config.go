package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	desktopConfigFileName = "config-v2.json"
	desktopDataDirName    = ".golang-cc"
	desktopConfigDirEnv   = "GOLANG_CC_DESKTOP_CONFIG_DIR"
)

const (
	defaultWindowWidth  = 1440
	defaultWindowHeight = 900
	minWindowWidth      = 1024
	minWindowHeight     = 700
)

type DesktopWindowGeometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (g DesktopWindowGeometry) valid() bool {
	return g.Width >= minWindowWidth && g.Height >= minWindowHeight
}

type windowState struct {
	Geometry   DesktopWindowGeometry `json:"geometry,omitempty"`
	Maximized  bool                  `json:"maximized,omitempty"`
	Fullscreen bool                  `json:"fullscreen,omitempty"`
}

func (s windowState) normalized() windowState {
	if !s.Geometry.valid() {
		s.Geometry = DesktopWindowGeometry{}
	}
	if s.Fullscreen {
		s.Maximized = false
	}
	return s
}

type desktopConfig struct {
	Workspace string      `json:"workspace,omitempty"`
	Window    windowState `json:"window,omitempty"`
}

func (c desktopConfig) normalized() desktopConfig {
	c.Window = c.Window.normalized()
	return c
}

func desktopDataDir() (string, error) {
	if root := strings.TrimSpace(os.Getenv(desktopConfigDirEnv)); root != "" {
		return filepath.Join(root, "golang-cc"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, desktopDataDirName), nil
}

func desktopConfigPath() (string, error) {
	dir, err := desktopDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, desktopConfigFileName), nil
}

func loadDesktopConfig() (desktopConfig, error) {
	path, err := desktopConfigPath()
	if err != nil {
		return desktopConfig{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return desktopConfig{}, nil
	}
	if err != nil {
		return desktopConfig{}, err
	}
	var config desktopConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return desktopConfig{}, err
	}
	config.Workspace = strings.TrimSpace(config.Workspace)
	if config.Workspace != "" {
		info, statErr := os.Stat(config.Workspace)
		if statErr != nil || !info.IsDir() {
			config.Workspace = ""
		}
	}
	return config.normalized(), nil
}

func saveDesktopConfig(config desktopConfig) error {
	path, err := desktopConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config.normalized(), "", "  ")
	if err != nil {
		return err
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func chooseWorkspace(ctx context.Context) (string, error) {
	workspace, err := wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
		Title: "选择 go-e2e 工作目录",
	})
	if err != nil {
		return "", err
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", errors.New("workspace selection cancelled")
	}
	info, err := os.Stat(workspace)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("selected workspace is not a directory")
	}
	return filepath.Clean(workspace), nil
}
