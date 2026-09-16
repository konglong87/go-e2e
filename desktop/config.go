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

const desktopConfigFileName = "config.json"

type desktopConfig struct {
	Workspace string `json:"workspace,omitempty"`
}

func desktopConfigPath() (string, error) {
	if root := strings.TrimSpace(os.Getenv("GOLANG_CC_DESKTOP_CONFIG_DIR")); root != "" {
		return filepath.Join(root, "golang-cc", desktopConfigFileName), nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "golang-cc", desktopConfigFileName), nil
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
	return config, nil
}

func saveDesktopConfig(config desktopConfig) error {
	path, err := desktopConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
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
		Title: "选择 golang-cc 工作目录",
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
