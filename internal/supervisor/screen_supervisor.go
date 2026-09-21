package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/konglong87/go-e2e/internal/provisioning"
)

type ScreenSupervisor struct {
	ScriptPath string
	StateDir   string
	Runner     CommandRunner
	BaseEnv    []string
}

func (s ScreenSupervisor) List(ctx context.Context) ([]provisioning.WorkerStatus, error) {
	stateDir := s.StateDir
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve channel worker state directory: %w", err)
		}
		stateDir = provisioning.DefaultChannelWorkerStateDir(home)
	}
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []provisioning.WorkerStatus{}, nil
		}
		return nil, err
	}
	workers := make([]provisioning.WorkerStatus, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".env") {
			continue
		}
		workerName := strings.TrimSuffix(entry.Name(), ".env")
		values, parseErr := readWorkerEnv(filepath.Join(stateDir, entry.Name()))
		if parseErr != nil {
			continue
		}
		spec := provisioning.WorkerSpec{
			WorkerName:  workerName,
			AccountKey:  workerEnvValue(values, "CHANNEL_ACCOUNT_KEY"),
			Provider:    workerEnvValue(values, "CHANNEL_MODEL_PROVIDER"),
			Model:       workerEnvValue(values, "CHANNEL_MODEL"),
			SettingsRef: workerEnvValue(values, "CHANNEL_SETTINGS_FILE"),
			Environment: values,
		}
		status, statusErr := s.run(ctx, "status", spec)
		if statusErr != nil && status.State == provisioning.WorkerStateUnknown {
			status.State = provisioning.WorkerStateFailed
			status.Message = statusErr.Error()
		}
		status.AccountKey = workerEnvValue(values, "CHANNEL_ACCOUNT_KEY")
		status.TenantID, _ = strconv.ParseUint(workerEnvValue(values, "CHANNEL_TENANT_ID"), 10, 64)
		status.AccountID, _ = strconv.ParseUint(workerEnvValue(values, "CHANNEL_ACCOUNT_ID"), 10, 64)
		workers = append(workers, status)
	}
	return workers, nil
}

func workerEnvValue(values map[string]string, suffix string) string {
	if value := strings.TrimSpace(values["GO_E2E_"+suffix]); value != "" {
		return value
	}
	return strings.TrimSpace(values["GOLANG_CC_"+suffix])
}

func (s ScreenSupervisor) Start(ctx context.Context, spec provisioning.WorkerSpec) (provisioning.WorkerStatus, error) {
	return s.run(ctx, "start", spec)
}
func (s ScreenSupervisor) Restart(ctx context.Context, spec provisioning.WorkerSpec) (provisioning.WorkerStatus, error) {
	return s.run(ctx, "restart", spec)
}
func (s ScreenSupervisor) Stop(ctx context.Context, spec provisioning.WorkerSpec) (provisioning.WorkerStatus, error) {
	return s.run(ctx, "stop", spec)
}
func (s ScreenSupervisor) Status(ctx context.Context, spec provisioning.WorkerSpec) (provisioning.WorkerStatus, error) {
	return s.run(ctx, "status", spec)
}
func (s ScreenSupervisor) Logs(ctx context.Context, spec provisioning.WorkerSpec, tail int) (string, error) {
	if tail <= 0 {
		tail = 100
	}
	status, err := s.Status(ctx, spec)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(status.LogPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	return strings.Join(lines, "\n"), nil
}

func (s ScreenSupervisor) run(ctx context.Context, action string, spec provisioning.WorkerSpec) (provisioning.WorkerStatus, error) {
	if strings.TrimSpace(spec.AccountKey) == "" {
		return provisioning.WorkerStatus{State: provisioning.WorkerStateFailed, ObservedAt: time.Now().UTC()}, fmt.Errorf("account key is required")
	}
	runner := s.Runner
	if runner == nil {
		runner = osCommandRunner{}
	}
	script := s.ScriptPath
	if script == "" {
		script = "scripts/channel-worker-screen.sh"
	}
	env := append([]string{}, s.BaseEnv...)
	for key, value := range spec.Environment {
		env = append(env, key+"="+value)
	}
	workerName := spec.WorkerName
	if workerName == "" {
		workerName = spec.AccountKey
	}
	spec.WorkerName = workerName
	env = append(env, "GOLANG_CC_CHANNEL_WORKER_NAME="+workerName, "GOLANG_CC_CHANNEL_ACCOUNT_KEY="+spec.AccountKey, "GOLANG_CC_CHANNEL_MODEL_PROVIDER="+spec.Provider, "GOLANG_CC_CHANNEL_MODEL="+spec.Model, "GOLANG_CC_CHANNEL_SETTINGS_FILE="+spec.SettingsRef)
	output, err := runner.Run(ctx, script, []string{action}, env)
	status := parseStatus(string(output), spec)
	if err != nil {
		status.State = provisioning.WorkerStateFailed
		status.Message = strings.TrimSpace(string(output))
		return status, err
	}
	return status, nil
}

func parseStatus(output string, spec provisioning.WorkerSpec) provisioning.WorkerStatus {
	status := provisioning.WorkerStatus{State: provisioning.WorkerStateUnknown, Provider: spec.Provider, Model: spec.Model, AccountKey: spec.AccountKey, ObservedAt: time.Now().UTC(), Screen: "golang-cc-channel-" + spec.WorkerName}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "running:"):
			status.State = provisioning.WorkerStateRunning
		case strings.HasPrefix(line, "degraded:"):
			status.State = provisioning.WorkerStateDegraded
		case strings.HasPrefix(line, "stopped:"):
			status.State = provisioning.WorkerStateStopped
		case strings.HasPrefix(line, "log:"):
			status.LogPath = strings.TrimSpace(strings.TrimPrefix(line, "log:"))
		case strings.HasPrefix(line, "pids:"):
			fields := strings.Fields(strings.TrimPrefix(line, "pids:"))
			if len(fields) > 0 {
				status.PID, _ = strconv.Atoi(fields[0])
			}
		}
	}
	return status
}

func readWorkerEnv(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
		if key != "" {
			values[key] = value
		}
	}
	return values, nil
}

type osCommandRunner struct{}

func (osCommandRunner) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	return cmd.CombinedOutput()
}

var _ provisioning.WorkerSupervisor = ScreenSupervisor{}
