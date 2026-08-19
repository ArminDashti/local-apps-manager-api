package runner

import (
	"context"

	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
)

// NativeRunner starts/stops via Run-ListedApps.ps1.
type NativeRunner struct {
	scriptPath string
	configPath string
}

func NewNativeRunner(scriptPath, configPath string) *NativeRunner {
	return &NativeRunner{scriptPath: scriptPath, configPath: configPath}
}

func (n *NativeRunner) Start(_ context.Context, pair discover.Pair) error {
	args := []string{"-Name", pair.Stem, "-SkipStopBeforeStart"}
	if n.configPath != "" {
		args = append([]string{"-Config", n.configPath}, args...)
	}
	return runPowerShell(n.scriptPath, args)
}

func (n *NativeRunner) Stop(_ context.Context, pair discover.Pair) error {
	args := []string{"-StopName", pair.Stem}
	if n.configPath != "" {
		args = append([]string{"-Config", n.configPath}, args...)
	}
	return runPowerShell(n.scriptPath, args)
}
