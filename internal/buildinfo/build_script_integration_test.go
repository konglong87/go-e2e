//go:build integration

package buildinfo_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konglong87/go-e2e/internal/buildinfo"
)

func TestBuildScriptInjectsReproducibleIdentity(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "golang-cc")
	command := exec.Command("bash", "scripts/build.sh")
	command.Dir = repositoryRoot
	command.Env = append(os.Environ(),
		"GOTOOLCHAIN=local",
		"VERSION=v-build-test",
		"SOURCE_DATE_EPOCH=1786417445",
		"OUTPUT="+binary,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("scripts/build.sh failed: %v\n%s", err, output)
	}

	jsonOutput, err := exec.Command(binary, "version", "--json").Output()
	if err != nil {
		t.Fatal(err)
	}
	var identity buildinfo.Info
	if err := json.Unmarshal(jsonOutput, &identity); err != nil {
		t.Fatalf("decode build identity: %v\n%s", err, jsonOutput)
	}
	if identity.Version != "v-build-test" || identity.BuildTime != "2026-08-11T03:04:05Z" || identity.Revision == "" || !identity.DirtyKnown || identity.GoToolchain == "" {
		t.Fatalf("built identity = %+v", identity)
	}

	textOutput, err := exec.Command(binary, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(textOutput); got != "v-build-test (golang-cc)\n" {
		t.Fatalf("--version = %q", got)
	}
	if strings.Contains(string(jsonOutput), repositoryRoot) {
		t.Fatalf("build identity leaked repository path: %s", jsonOutput)
	}
}
