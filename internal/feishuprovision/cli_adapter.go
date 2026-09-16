package feishuprovision

import (
	"context"
	"os/exec"

	"github.com/konglong87/go-e2e/internal/provisioning"
)

type CLIDiscovery struct{ LookPath func(string) (string, error) }

func (d CLIDiscovery) DiscoverCLI(ctx context.Context) (provisioning.CLIAvailability, error) {
	lookup := d.LookPath
	if lookup == nil {
		lookup = exec.LookPath
	}
	result := provisioning.CLIAvailability{}
	if _, err := lookup("golang-cc"); err == nil {
		result.ProjectCLI = true
	}
	if _, err := lookup("lark-cli"); err == nil {
		result.LarkCLI = true
	}
	if !result.ProjectCLI && !result.LarkCLI {
		result.Message = "no Feishu onboarding CLI found; select an existing account or enter credentials manually"
	}
	return result, nil
}
