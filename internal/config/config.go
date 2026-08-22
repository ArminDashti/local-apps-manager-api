package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	JWTSecret           string
	DefaultUsername     string
	DefaultPassword     string
	CORSOrigins         []string
	GitHubRoot          string
	DockerStatePath     string
	DockerRunnerScript  string
	NativeRunnerScript  string
	NativeStatePath     string
	NativeAppsConfig       string
	NativeHotReloadScript  string
	ServerSSHTimeoutSec    int
	HostIP                 string
	PcArminBase            string
}

func Load() Config {
	return Config{
		HTTPAddr:            getenv("HTTP_ADDR", "127.0.0.1:8195"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://localapps:localapps@127.0.0.1:5455/localapps?sslmode=disable"),
		JWTSecret:           getenv("JWT_SECRET", "change-me-in-production"),
		DefaultUsername:     getenv("DEFAULT_USERNAME", "armin"),
		DefaultPassword:     getenv("DEFAULT_PASSWORD", "dopadopa123"),
		CORSOrigins:         splitCSV(getenv("CORS_ORIGINS", "http://127.0.0.1:5195,http://localhost:5195")),
		GitHubRoot:          getenv("GITHUB_ROOT", "C:/Users/armin/GitHub"),
		DockerStatePath:     getenv("DOCKER_STATE_PATH", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/run-apps-on-local-docker/.run-local-docker-pairs-state.json"),
		DockerRunnerScript:  getenv("DOCKER_RUNNER_SCRIPT", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/run-apps-on-local-docker/scripts/Run-LocalDockerAppPairs.ps1"),
		NativeRunnerScript:  getenv("NATIVE_RUNNER_SCRIPT", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/run-all-apps-locally/scripts/Run-ListedApps.ps1"),
		NativeStatePath:     getenv("NATIVE_STATE_PATH", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/run-all-apps-locally/.run-listed-apps-state.json"),
		NativeAppsConfig:    getenv("NATIVE_APPS_CONFIG", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/run-all-apps-locally/apps.yaml"),
		ServerSSHTimeoutSec: getenvInt("SERVER_SSH_TIMEOUT_SEC", 300),
		HostIP:                 getenv("HOST_IP", "127.0.0.1"),
		NativeHotReloadScript:  getenv("NATIVE_HOT_RELOAD_SCRIPT", "C:/Users/armin/.cursor/plugins/local/devops-by-armin/skills/local-hot-reload/scripts/Run-LocalHotReloadPair.ps1"),
		PcArminBase:            getenv("PC_ARMIN_BASE", "http://pc-armin"),
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

