package dockerstate

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type StateFile struct {
	StartedAt string     `json:"startedAt"`
	Root      string     `json:"root"`
	Pairs     []StateRow `json:"pairs"`
}

type StateRow struct {
	Stem            string   `json:"stem"`
	Status          string   `json:"status"`
	Reason          *string  `json:"reason"`
	ApiDir          string   `json:"apiDir"`
	WebUiDir        string   `json:"webuiDir"`
	ApiStack        string   `json:"apiStack"`
	WebUiStack      *string  `json:"webuiStack"`
	ApiURL          string   `json:"apiUrl"`
	WebUiURL        string   `json:"webuiUrl"`
	ApiHostPort     int      `json:"apiHostPort"`
	WebUiHostPort   int      `json:"webuiHostPort"`
	PostgresHostPort *int    `json:"postgresHostPort"`
}

func ReadState(path string) (*StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &StateFile{Pairs: []StateRow{}}, nil
		}
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var state StateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return &StateFile{Pairs: []StateRow{}}, nil
	}
	return &state, nil
}

func RowByStem(state *StateFile, stem string) *StateRow {
	if state == nil {
		return nil
	}
	for i := range state.Pairs {
		if strings.EqualFold(state.Pairs[i].Stem, stem) {
			return &state.Pairs[i]
		}
	}
	return nil
}

func RunningProjects() (map[string]bool, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.Label \"com.docker.compose.project\"}}").Output()
	if err != nil {
		return nil, err
	}
	projects := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			projects[line] = true
		}
	}
	return projects, nil
}

func LocalProjectName(stem string) string {
	return stem + "-local"
}

func OnDocker(stem, apiStack, webUiStack string, projects map[string]bool) bool {
	if projects[LocalProjectName(stem)] {
		return true
	}
	if projects[apiStack] {
		return true
	}
	if webUiStack != "" && projects[webUiStack] {
		return true
	}
	// fallback: any container name containing stem
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}").Output()
	if err != nil {
		return false
	}
	needle := strings.ToLower(stem)
	for _, name := range strings.Split(string(out), "\n") {
		if strings.Contains(strings.ToLower(name), needle) {
			return true
		}
	}
	return false
}


// HostPortsForProject returns published host ports for api/webui-ish services in a compose project.
func HostPortsForProject(project string) (apiPort, webuiPort int) {
	if project == "" {
		return 0, 0
	}
	out, err := exec.Command(
		"docker", "ps",
		"--filter", "label=com.docker.compose.project="+project,
		"--format", "{{.Label \"com.docker.compose.service\"}}\t{{.Ports}}",
	).Output()
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		svc := strings.ToLower(parts[0])
		hostPort := firstHostPort(parts[1])
		if hostPort == 0 {
			continue
		}
		switch {
		case strings.Contains(svc, "web"), strings.Contains(svc, "ui"), strings.Contains(svc, "frontend"), svc == "vite":
			if webuiPort == 0 {
				webuiPort = hostPort
			}
		case strings.Contains(svc, "api"), strings.Contains(svc, "backend"), strings.Contains(svc, "server"), svc == "app":
			if apiPort == 0 {
				apiPort = hostPort
			}
		}
	}
	return apiPort, webuiPort
}

func firstHostPort(portsField string) int {
	for _, chunk := range strings.Split(portsField, ", ") {
		chunk = strings.TrimSpace(chunk)
		if !strings.Contains(chunk, "->") {
			continue
		}
		left := strings.Split(chunk, "->")[0]
		if i := strings.LastIndex(left, ":"); i >= 0 {
			if n, err := strconv.Atoi(left[i+1:]); err == nil {
				return n
			}
		}
	}
	return 0
}
