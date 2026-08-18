package discover

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var webUiSuffixes = []string{"-webui", "-web", "-ui"}
var webUiServiceRe = regexp.MustCompile(`(?m)^\s{2}webui\s*:`)

type Pair struct {
	Stem              string
	ApiDir            string
	WebUiDir          string
	WebUiName         string
	ApiCompose        string
	WebUiCompose      string
	Combined          bool
	SkipReason        string
	ApiStack          string
	WebUiStack        string
	ApiInternalPort   int
	WebUiInternalPort int
}

func FindPairs(root string) ([]Pair, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	byName := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			byName[e.Name()] = filepath.Join(root, e.Name())
		}
	}

	var pairs []Pair
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "-api") {
			continue
		}
		stem := strings.TrimSuffix(name, "-api")
		uiName := ""
		var uiDir string
		for _, suffix := range webUiSuffixes {
			candidate := stem + suffix
			if dir, ok := byName[candidate]; ok {
				uiName = candidate
				uiDir = dir
				break
			}
		}
		apiDir := byName[name]
		p := Pair{Stem: stem, ApiDir: apiDir, WebUiDir: uiDir, WebUiName: uiName}
		if uiDir == "" {
			p.SkipReason = "no WebUI sibling"
		} else {
			resolveProductionPlan(&p)
		}
		p.ApiStack = stackName(apiDir, stem+"-api")
		p.WebUiStack = stackName(uiDir, stem+"-webui")
		if p.Combined {
			p.WebUiStack = p.ApiStack
		}
		p.ApiInternalPort = resolveInternalPort(p.ApiDir, 0)
		if p.WebUiDir != "" {
			p.WebUiInternalPort = resolveInternalPort(p.WebUiDir, 80)
		}
		pairs = append(pairs, p)
	}
	return pairs, nil
}

func resolveProductionPlan(p *Pair) {
	apiBase := findBaseCompose(p.ApiDir)
	uiBase := findBaseCompose(p.WebUiDir)
	if apiBase == "" {
		p.SkipReason = "API production docker-compose missing"
		return
	}
	p.ApiCompose = apiBase
	if hasWebUiService(apiBase) {
		p.Combined = true
		return
	}
	if uiBase == "" {
		p.SkipReason = "WebUI production docker-compose missing"
		return
	}
	p.WebUiCompose = uiBase
}

func findBaseCompose(dir string) string {
	if dir == "" {
		return ""
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func hasWebUiService(composePath string) bool {
	data, err := os.ReadFile(composePath)
	if err != nil {
		return false
	}
	return webUiServiceRe.Match(data)
}

func stackName(projectDir, fallback string) string {
	if projectDir == "" {
		return fallback
	}
	yamlPath := filepath.Join(projectDir, ".armin", "docker-scripts", "run-on-docker-local.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "stack_name:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "stack_name:"))
			val = strings.Trim(val, `"'`)
			if val != "" {
				return val
			}
		}
	}
	return fallback
}

func StackLabel(p Pair) string {
	if p.Combined {
		return p.ApiStack
	}
	if p.WebUiStack != "" && p.WebUiStack != p.ApiStack {
		return p.ApiStack + " / " + p.WebUiStack
	}
	return p.ApiStack
}
