package runner

import (
	"context"
	"os"

	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
)

// NativeRunner starts/stops host processes via local-hot-reload script when present,
// otherwise via run-all-apps-locally Run-ListedApps.ps1.
type NativeRunner struct {
	scriptPath     string
	configPath     string
	hotReloadPath  string
}

func NewNativeRunner(scriptPath, configPath, hotReloadPath string) *NativeRunner {
	return &NativeRunner{scriptPath: scriptPath, configPath: configPath, hotReloadPath: hotReloadPath}
}

func (n *NativeRunner) Start(_ context.Context, pair discover.Pair) error {
	if n.hotReloadPath != "" {
		if _, err := os.Stat(n.hotReloadPath); err == nil {
			return runPowerShell(n.hotReloadPath, []string{
				"-Root", pair.ApiDir, // unused by script when -Name set; script resolves siblings
				"-Name", pair.Stem,
				"-SkipStopBeforeStart",
			})
		}
	}
	args := []string{"-Name", pair.Stem, "-SkipStopBeforeStart"}
	if n.configPath != "" {
		args = append([]string{"-Config", n.configPath}, args...)
	}
	return runPowerShell(n.scriptPath, args)
}

func (n *NativeRunner) Stop(_ context.Context, pair discover.Pair) error {
	if n.hotReloadPath != "" {
		if _, err := os.Stat(n.hotReloadPath); err == nil {
			return runPowerShell(n.hotReloadPath, []string{"-StopName", pair.Stem})
		}
	}
	args := []string{"-StopName", pair.Stem}
	if n.configPath != "" {
		args = append([]string{"-Config", n.configPath}, args...)
	}
	return runPowerShell(n.scriptPath, args)
}
