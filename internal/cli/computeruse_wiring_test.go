package cli

import (
	"context"
	"testing"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
	"github.com/konglong87/go-e2e/internal/config"
	"github.com/konglong87/go-e2e/internal/runtimeprofile"
)

type computerUseWiringService struct{}

func (computerUseWiringService) Capabilities(context.Context, cu.SessionOwner, string) (cu.Capabilities, error) {
	return cu.Capabilities{}, nil
}
func (computerUseWiringService) Observe(context.Context, cu.SessionOwner, cu.ObserveRequest) (cu.Observation, error) {
	return cu.Observation{}, nil
}
func (computerUseWiringService) Execute(context.Context, cu.SessionOwner, cu.Action) (cu.ActionReceipt, error) {
	return cu.ActionReceipt{}, nil
}
func (computerUseWiringService) Pause(context.Context, cu.SessionOwner, string) error  { return nil }
func (computerUseWiringService) Resume(context.Context, cu.SessionOwner, string) error { return nil }
func (computerUseWiringService) Stop(context.Context, cu.SessionOwner, string) error   { return nil }

func TestComputerUseRegistrationIsOptInAndImageGated(t *testing.T) {
	service := computerUseWiringService{}
	cases := []struct {
		name string
		opts options
		want bool
	}{
		{name: "default", opts: options{}, want: false},
		{name: "missing service", opts: options{computerUseProfile: true, computerUseImageSupported: true, tenantID: 7, tenantUserID: 11}, want: false},
		{name: "missing image support", opts: options{computerUseProfile: true, computerUseService: service, tenantID: 7, tenantUserID: 11}, want: false},
		{name: "bare profile", opts: options{runtimeProfile: runtimeprofile.ProfileBare, computerUseProfile: true, computerUseService: service, computerUseImageSupported: true, tenantID: 7, tenantUserID: 11}, want: false},
		{name: "explicit desktop capability", opts: options{computerUseProfile: true, computerUseService: service, computerUseImageSupported: true, tenantID: 7, tenantUserID: 11}, want: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			registered := coreRuntimeToolsWithOptions(config.Settings{}, nil, "test-model", test.opts)
			found := false
			for _, tool := range registered {
				if tool.Name() == "ComputerUse" {
					found = true
				}
			}
			if found != test.want {
				t.Fatalf("ComputerUse registered=%v, want %v", found, test.want)
			}
		})
	}
}
