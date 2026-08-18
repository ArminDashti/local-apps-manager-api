package dockerstate

import (
	"encoding/json"
	"os"
	"os/exec"
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
	var state StateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
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

func OnDocker(stem, apiStack, webUiStack string, projects map[string]bool) bool {
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
