package serverstate

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type DeployConfig struct {
	StackName    string
	SSH          string
	VolumeDir    string
	ComposeFile  string
	InternalPort string
	PublicHint   string // optional comment-derived or empty
}

var placeholderRe = regexp.MustCompile(`<[^>]+>`)

func ReadDeployConfig(projectDir string) (*DeployConfig, error) {
	yamlPath := filepath.Join(projectDir, ".armin", "docker-scripts", "run-on-docker-server.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}
	cfg := &DeployConfig{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch key {
		case "stack_name":
			cfg.StackName = val
		case "ssh":
			cfg.SSH = val
		case "volume_dir":
			cfg.VolumeDir = val
		case "compose_file":
			cfg.ComposeFile = val
		case "internal_port":
			cfg.InternalPort = val
		}
	}
	if cfg.StackName == "" {
		return nil, fmt.Errorf("stack_name missing in %s", yamlPath)
	}
	return cfg, nil
}

func HasValidServerDeploy(projectDir string) bool {
	script := filepath.Join(projectDir, ".armin", "docker-scripts", "run-on-docker-server.ps1")
	yamlPath := filepath.Join(projectDir, ".armin", "docker-scripts", "run-on-docker-server.yaml")
	if _, err := os.Stat(script); err != nil {
		return false
	}
	cfg, err := ReadDeployConfig(projectDir)
	if err != nil {
		return false
	}
	_ = yamlPath
	if cfg.SSH == "" || placeholderRe.MatchString(cfg.SSH) {
		return false
	}
	if cfg.VolumeDir == "" || placeholderRe.MatchString(cfg.VolumeDir) {
		return false
	}
	return true
}

func ScriptPath(projectDir string) string {
	return filepath.Join(projectDir, ".armin", "docker-scripts", "run-on-docker-server.ps1")
}

// OnServer checks whether a compose project is running on the remote host via SSH.
func OnServer(cfg *DeployConfig, timeoutSec int) bool {
	if cfg == nil || cfg.SSH == "" || cfg.StackName == "" {
		return false
	}
	cmdArgs, err := sshCommandArgs(cfg.SSH, fmt.Sprintf(
		`docker ps --filter "label=com.docker.compose.project=%s" --format "{{.Names}}"`,
		cfg.StackName,
	))
	if err != nil {
		return false
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			return false
		}
		return strings.TrimSpace(out.String()) != ""
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		_ = cmd.Process.Kill()
		return false
	}
}

// PublicURL builds a best-effort HTTPS URL from stack name (HAProxy convention).
func PublicURL(stackName string) string {
	if stackName == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.xaigrok.ir/", stackName)
}

func sshCommandArgs(sshValue, remoteCommand string) ([]string, error) {
	sshValue = strings.TrimSpace(sshValue)
	if strings.HasPrefix(strings.ToLower(sshValue), "ssh ") {
		parts := strings.Fields(sshValue)
		// e.g. ssh t3 -p 80
		args := append([]string{}, parts...)
		args = append(args, remoteCommand)
		return args, nil
	}
	// host@user@password — not supported for silent status (needs sshpass)
	if strings.Count(sshValue, "@") >= 2 {
		return nil, fmt.Errorf("password SSH mode not supported for status probe")
	}
	return []string{"ssh", sshValue, remoteCommand}, nil
}
