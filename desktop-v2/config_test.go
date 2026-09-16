package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopConfigRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOLANG_CC_DESKTOP_CONFIG_DIR", root)
	workspace := filepath.Join(root, "project")
	if err := os.Mkdir(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := saveDesktopConfig(desktopConfig{Workspace: workspace}); err != nil {
		t.Fatal(err)
	}
	got, err := loadDesktopConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Workspace != workspace {
		t.Fatalf("workspace = %q, want %q", got.Workspace, workspace)
	}
	info, err := os.Stat(filepath.Join(root, "golang-cc", desktopConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestLoadDesktopConfigClearsMissingWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOLANG_CC_DESKTOP_CONFIG_DIR", root)
	if err := saveDesktopConfig(desktopConfig{Workspace: filepath.Join(root, "missing")}); err != nil {
		t.Fatal(err)
	}
	got, err := loadDesktopConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Workspace != "" {
		t.Fatalf("workspace = %q, want empty", got.Workspace)
	}
}
