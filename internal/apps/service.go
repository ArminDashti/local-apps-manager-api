package apps

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ArminDashti/local-apps-manager-api/internal/config"
	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
	"github.com/ArminDashti/local-apps-manager-api/internal/dockerstate"
	"github.com/ArminDashti/local-apps-manager-api/internal/hostip"
	"github.com/ArminDashti/local-apps-manager-api/internal/probe"
	"github.com/ArminDashti/local-apps-manager-api/internal/runner"
	"github.com/ArminDashti/local-apps-manager-api/internal/store"
)

type Row struct {
	Stem              string  `json:"stem"`
	Stack             string  `json:"stack"`
	App               string  `json:"app"`
	ApiApp            string  `json:"apiApp"`
	WebUiApp          string  `json:"webuiApp"`
	Host              string  `json:"host"`
	ExternalHost      string  `json:"externalHost"`
	ApiPort           int     `json:"apiPort"`
	WebUiPort         int     `json:"webuiPort"`
	ApiInternalPort   int     `json:"apiInternalPort"`
	WebUiInternalPort int     `json:"webuiInternalPort"`
	ApiURL            string  `json:"apiUrl"`
	WebUiURL          string  `json:"webuiUrl"`
	OnDocker          bool    `json:"onDocker"`
	Status            string  `json:"status"`
	Enabled           bool    `json:"enabled"`
	SkipReason        *string `json:"skipReason"`
	ActionInProgress  bool    `json:"actionInProgress"`
}

type Service struct {
	cfg    config.Config
	store  *store.Store
	runner *runner.Runner
}

func NewService(cfg config.Config, st *store.Store, run *runner.Runner) *Service {
	return &Service{cfg: cfg, store: st, runner: run}
}

func (s *Service) List(ctx context.Context) ([]Row, error) {
	pairs, err := discover.FindPairs(s.cfg.GitHubRoot)
	if err != nil {
		return nil, err
	}
	state, err := dockerstate.ReadState(s.cfg.DockerStatePath)
	if err != nil {
		return nil, err
	}
	projects, err := dockerstate.RunningProjects()
	if err != nil {
		projects = map[string]bool{}
	}
	prefs, err := s.store.ListAppPreferences(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(pairs))
	for _, p := range pairs {
		row := s.buildRow(p, state, projects, prefs)
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *Service) buildRow(p discover.Pair, state *dockerstate.StateFile, projects map[string]bool, prefs map[string]bool) Row {
	st := dockerstate.RowByStem(state, p.Stem)
	apiPort, webuiPort := 0, 0
	apiURL, webuiURL := "", ""
	if st != nil {
		apiPort = st.ApiHostPort
		webuiPort = st.WebUiHostPort
		apiURL = st.ApiURL
		webuiURL = st.WebUiURL
	}
	onDocker := dockerstate.OnDocker(p.Stem, p.ApiStack, p.WebUiStack, projects)
	status := "Down"
	if webuiPort > 0 && probe.PortListening(s.cfg.HostIP, webuiPort) {
		status = "UP"
	} else if apiPort > 0 && probe.PortListening(s.cfg.HostIP, apiPort) && webuiPort == 0 {
		status = "UP"
	}
	enabled := onDocker || status == "UP"
	if v, ok := prefs[p.Stem]; ok {
		enabled = v
	}
	var skip *string
	if p.SkipReason != "" {
		skip = &p.SkipReason
	}
	externalHost := hostip.Resolve(s.cfg.HostIP)
	if apiURL == "" && apiPort > 0 {
		apiURL = fmt.Sprintf("http://%s:%d/", externalHost, apiPort)
	}
	if webuiURL == "" && webuiPort > 0 {
		webuiURL = fmt.Sprintf("http://%s:%d/", externalHost, webuiPort)
	}
	apiApp := filepath.Base(p.ApiDir)
	if apiApp == "" || apiApp == "." {
		apiApp = p.Stem + "-api"
	}
	return Row{
		Stem:              p.Stem,
		Stack:             p.Stem,
		App:               p.Stem,
		ApiApp:            apiApp,
		WebUiApp:          p.WebUiName,
		Host:              s.cfg.HostIP,
		ExternalHost:      externalHost,
		ApiPort:           apiPort,
		WebUiPort:         webuiPort,
		ApiInternalPort:   p.ApiInternalPort,
		WebUiInternalPort: p.WebUiInternalPort,
		ApiURL:            apiURL,
		WebUiURL:          webuiURL,
		OnDocker:          onDocker,
		Status:            status,
		Enabled:           enabled,
		SkipReason:        skip,
		ActionInProgress:  s.runner.IsRunning(p.Stem),
	}
}

func (s *Service) SetEnabled(ctx context.Context, stem string, enabled bool) error {
	pairs, err := discover.FindPairs(s.cfg.GitHubRoot)
	if err != nil {
		return err
	}
	var pair *discover.Pair
	for i := range pairs {
		if pairs[i].Stem == stem {
			pair = &pairs[i]
			break
		}
	}
	if pair == nil {
		return fmt.Errorf("app not found: %s", stem)
	}
	if pair.SkipReason != "" {
		return fmt.Errorf("cannot toggle %s: %s", stem, pair.SkipReason)
	}
	if s.runner.IsRunning(stem) {
		return fmt.Errorf("action already in progress for %s", stem)
	}
	if err := s.store.SetAppEnabled(ctx, stem, enabled); err != nil {
		return err
	}
	if enabled {
		return s.runner.Start(stem)
	}
	return s.runner.Stop(stem)
}
