package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
	"github.com/ArminDashti/local-apps-manager-api/internal/dockerstate"
)

// DockerRunner starts/stops production-sim stacks, or local-docker-hot-reload compose when present.
type DockerRunner struct {
	scriptPath string
	root       string
}

func NewDockerRunner(scriptPath, root string) *DockerRunner {
	return &DockerRunner{scriptPath: scriptPath, root: root}
}

func (d *DockerRunner) Start(_ context.Context, pair discover.Pair) error {
	if pair.LocalCompose != "" {
		return runLocalCompose(pair, true)
	}
	return runPowerShell(d.scriptPath, []string{
		"-Root", d.root,
		"-Name", pair.Stem,
		"-SkipStopBeforeStart",
	})
}

func (d *DockerRunner) Stop(_ context.Context, pair discover.Pair) error {
	if pair.LocalCompose != "" {
		return runLocalCompose(pair, false)
	}
	// Also try hot-reload project stop in case it was started outside the manager.
	_ = runLocalCompose(pair, false)
	return runPowerShell(d.scriptPath, []string{
		"-Root", d.root,
		"-StopName", pair.Stem,
	})
}

func runLocalCompose(pair discover.Pair, up bool) error {
	compose := pair.LocalCompose
	if compose == "" {
		compose = findComposeLocal(pair.ApiDir)
	}
	if compose == "" {
		if up {
			return fmt.Errorf("no docker-compose.local.yml for %s", pair.Stem)
		}
		return nil
	}
	project := dockerstate.LocalProjectName(pair.Stem)
	args := []string{"compose", "-f", compose, "-p", project}
	if up {
		args = append(args, "up", "-d", "--force-recreate")
	} else {
		args = append(args, "down")
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = filepath.Dir(compose)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker compose %s: %s", pair.Stem, msg)
	}
	return nil
}

func findComposeLocal(apiDir string) string {
	for _, name := range []string{
		"docker-compose.local.yml",
		"docker-compose.local.yaml",
		"compose.local.yml",
		"compose.local.yaml",
	} {
		path := filepath.Join(apiDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
