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

func TestDesktopDataDirDefaultsToGoE2EOwnedHomeDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("GOLANG_CC_DESKTOP_CONFIG_DIR", "")
	got, err := desktopDataDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".golang-cc")
	if got != want {
		t.Fatalf("desktop data dir = %q, want %q", got, want)
	}
}

func TestDesktopConfigRoundTripPreservesWindowState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOLANG_CC_DESKTOP_CONFIG_DIR", root)
	workspace := filepath.Join(root, "project")
	if err := os.Mkdir(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	want := desktopConfig{
		Workspace: workspace,
		Window: windowState{
			Geometry:  DesktopWindowGeometry{X: 120, Y: 80, Width: 1280, Height: 760},
			Maximized: true,
		},
	}
	if err := saveDesktopConfig(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadDesktopConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
}

func TestLoadDesktopConfigDropsInvalidWindowGeometry(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOLANG_CC_DESKTOP_CONFIG_DIR", root)
	if err := saveDesktopConfig(desktopConfig{
		Window: windowState{
			Geometry:   DesktopWindowGeometry{X: 10, Y: 20, Width: 640, Height: 480},
			Fullscreen: true,
			Maximized:  true,
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := loadDesktopConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Window.Geometry != (DesktopWindowGeometry{}) {
		t.Fatalf("geometry = %#v, want zero geometry", got.Window.Geometry)
	}
	if !got.Window.Fullscreen || got.Window.Maximized {
		t.Fatalf("window state = %#v, want fullscreen only", got.Window)
	}
}

func TestNormalizeWorkspaceSelectionRequiresExistingDirectory(t *testing.T) {
	root := t.TempDir()
	got, err := normalizeWorkspaceSelection(filepath.Join(root, ".", "selected"))
	if err == nil {
		t.Fatal("normalizeWorkspaceSelection accepted a missing directory")
	}
	if got != "" {
		t.Fatalf("missing directory result = %q, want empty", got)
	}

	workspace := filepath.Join(root, "selected")
	if err := os.Mkdir(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = normalizeWorkspaceSelection(filepath.Join(workspace, "."))
	if err != nil {
		t.Fatalf("normalizeWorkspaceSelection() error = %v", err)
	}
	if got != workspace {
		t.Fatalf("normalized workspace = %q, want %q", got, workspace)
	}
}
