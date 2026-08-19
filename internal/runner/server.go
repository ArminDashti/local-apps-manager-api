package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
	"github.com/ArminDashti/local-apps-manager-api/internal/serverstate"
)

// ServerRunner deploys/tears down via per-repo run-on-docker-server.ps1.
type ServerRunner struct{}

func NewServerRunner() *ServerRunner {
	return &ServerRunner{}
}

func (s *ServerRunner) Start(_ context.Context, pair discover.Pair) error {
	dirs, err := serverDirs(pair)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		script := serverstate.ScriptPath(dir)
		if err := runPowerShell(script, nil); err != nil {
			return fmt.Errorf("server deploy %s: %w", dir, err)
		}
	}
	return nil
}

func (s *ServerRunner) Stop(_ context.Context, pair discover.Pair) error {
	dirs, err := serverDirs(pair)
	if err != nil {
		return err
	}
	var firstErr error
	for _, dir := range dirs {
		script := serverstate.ScriptPath(dir)
		// Prefer script -Stop when supported; fall back to SSH compose down.
		if err := runPowerShell(script, []string{"-Stop"}); err != nil {
			if fallbackErr := stopServerViaSSH(dir); fallbackErr != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("server stop %s: script=%v; ssh=%v", dir, err, fallbackErr)
				}
			}
		}
	}
	return firstErr
}

func stopServerViaSSH(projectDir string) error {
	cfg, err := serverstate.ReadDeployConfig(projectDir)
	if err != nil {
		return err
	}
	remote := fmt.Sprintf(
		"docker compose -p '%s' --project-directory '%s' down >/dev/null 2>&1 || docker compose -p '%s' down >/dev/null 2>&1 || true",
		cfg.StackName, cfg.VolumeDir, cfg.StackName,
	)
	args, err := sshInvokeArgs(cfg.SSH, remote)
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func sshInvokeArgs(sshValue, remoteCommand string) ([]string, error) {
	sshValue = strings.TrimSpace(sshValue)
	if strings.HasPrefix(strings.ToLower(sshValue), "ssh ") {
		parts := strings.Fields(sshValue)
		return append(parts, remoteCommand), nil
	}
	if strings.Count(sshValue, "@") >= 2 {
		return nil, fmt.Errorf("password SSH mode not supported for stop fallback")
	}
	return []string{"ssh", sshValue, remoteCommand}, nil
}

func serverDirs(pair discover.Pair) ([]string, error) {
	if !pair.HasServerDeploy {
		return nil, fmt.Errorf("server deploy scripts missing or invalid for %s", pair.Stem)
	}
	dirs := []string{pair.ApiDir}
	if !pair.Combined && pair.WebUiDir != "" && serverstate.HasValidServerDeploy(pair.WebUiDir) {
		dirs = append(dirs, pair.WebUiDir)
	}
	return dirs, nil
}
