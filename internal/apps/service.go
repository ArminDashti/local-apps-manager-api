package apps

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ArminDashti/local-apps-manager-api/internal/config"
	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
	"github.com/ArminDashti/local-apps-manager-api/internal/dockerstate"
	"github.com/ArminDashti/local-apps-manager-api/internal/hostip"
	"github.com/ArminDashti/local-apps-manager-api/internal/nativestate"
	"github.com/ArminDashti/local-apps-manager-api/internal/probe"
	"github.com/ArminDashti/local-apps-manager-api/internal/runner"
	"github.com/ArminDashti/local-apps-manager-api/internal/runmode"
	"github.com/ArminDashti/local-apps-manager-api/internal/serverstate"
	"github.com/ArminDashti/local-apps-manager-api/internal/store"
)

type Row struct {
	Stem              string  `json:"stem"`
	Stack             string  `json:"stack"`
	App               string  `json:"app"`
	ApiApp            string  `json:"apiApp"`
	WebUiApp          string  `json:"webuiApp"`
	ApiInternalPort   int     `json:"apiInternalPort"`
	WebUiInternalPort int     `json:"webuiInternalPort"`
	LocalEnabled      bool    `json:"localEnabled"`
	DockerEnabled     bool    `json:"dockerEnabled"`
	PublicEnabled     bool    `json:"publicEnabled"`
	LocalApiURL       string  `json:"localApiUrl"`
	LocalWebUiURL     string  `json:"localWebuiUrl"`
	LocalStatus       string  `json:"localStatus"`
	DockerApiURL      string  `json:"dockerApiUrl"`
	DockerWebUiURL    string  `json:"dockerWebuiUrl"`
	DockerStatus      string  `json:"dockerStatus"`
	PublicApiURL      string  `json:"publicApiUrl"`
	PublicWebUiURL    string  `json:"publicWebuiUrl"`
	PublicStatus      string  `json:"publicStatus"`
	HasServerDeploy   bool    `json:"hasServerDeploy"`
	OnLocal           bool    `json:"onLocal"`
	OnDocker          bool    `json:"onDocker"`
	OnServer          bool    `json:"onServer"`
	SkipReason        *string `json:"skipReason"`
	ActionInProgress  bool    `json:"actionInProgress"`
}

type UpdateRequest struct {
	Enabled *bool
	RunMode *runmode.Mode
}

type Service struct {
	cfg    config.Config
	store  *store.Store
	router *runner.Router
}

func NewService(cfg config.Config, st *store.Store, router *runner.Router) *Service {
	return &Service{cfg: cfg, store: st, router: router}
}

func (s *Service) List(ctx context.Context) ([]Row, error) {
	pairs, err := discover.FindPairs(s.cfg.GitHubRoot)
	if err != nil {
		return nil, err
	}
	dockerState, err := dockerstate.ReadState(s.cfg.DockerStatePath)
	if err != nil {
		return nil, err
	}
	nativeState, err := nativestate.ReadState(s.cfg.NativeStatePath)
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
		row := s.buildRow(p, dockerState, nativeState, projects, prefs)
		rows = append(rows, row)
	}
	return rows, nil
}

func modeUp(port int, host string, fallback bool) string {
	if port > 0 && probe.PortListening(host, port) {
		return "UP"
	}
	if fallback {
		return "UP"
	}
	return "Down"
}

func urlOrPort(host string, port int, existing string) string {
	if existing != "" {
		return existing
	}
	if port > 0 {
		return fmt.Sprintf("http://%s:%d/", host, port)
	}
	return ""
}

func (s *Service) buildRow(
	p discover.Pair,
	dockerState *dockerstate.StateFile,
	nativeState *nativestate.StateFile,
	projects map[string]bool,
	prefs map[string]store.AppPreference,
) Row {
	pref := store.AppPreference{}
	if v, ok := prefs[p.Stem]; ok {
		pref = v
	} else {
		// Infer from live state when no DB preference exists.
		pref.LocalEnabled = nativestate.OnLocal(nativeState, p.Stem)
		pref.DockerEnabled = dockerstate.OnDocker(p.Stem, p.ApiStack, p.WebUiStack, projects)
	}

	onDocker := dockerstate.OnDocker(p.Stem, p.ApiStack, p.WebUiStack, projects)
	onLocal := nativestate.OnLocal(nativeState, p.Stem)

	externalHost := hostip.Resolve(s.cfg.HostIP)

	// Local URLs
	localApiPort, localWebuiPort, localApiURL, localWebuiURL := nativestate.PortsForStem(nativeState, p.Stem)
	localApiURL = urlOrPort("127.0.0.1", localApiPort, localApiURL)
	localWebuiURL = urlOrPort("127.0.0.1", localWebuiPort, localWebuiURL)
	localStatus := "Down"
	if modeUp(localWebuiPort, "127.0.0.1", false) == "UP" || modeUp(localApiPort, "127.0.0.1", false) == "UP" {
		localStatus = "UP"
	}

	// Docker URLs
	dockerApiPort, dockerWebuiPort := 0, 0
	dockerApiURL, dockerWebuiURL := "", ""
	if st := dockerstate.RowByStem(dockerState, p.Stem); st != nil {
		dockerApiPort = st.ApiHostPort
		dockerWebuiPort = st.WebUiHostPort
		dockerApiURL = st.ApiURL
		dockerWebuiURL = st.WebUiURL
	}
	dockerApiURL = urlOrPort(externalHost, dockerApiPort, dockerApiURL)
	dockerWebuiURL = urlOrPort(externalHost, dockerWebuiPort, dockerWebuiURL)
	dockerStatus := "Down"
	if modeUp(dockerWebuiPort, s.cfg.HostIP, false) == "UP" ||
		modeUp(dockerApiPort, s.cfg.HostIP, dockerWebuiPort == 0) == "UP" ||
		onDocker {
		dockerStatus = "UP"
	}

	// Public URLs
	publicApiURL, publicWebuiURL := "", ""
	onServer := false
	if p.HasServerDeploy {
		if cfg, err := serverstate.ReadDeployConfig(p.ApiDir); err == nil {
			publicApiURL = serverstate.PublicURL(cfg.StackName)
			if serverstate.OnServer(cfg, 5) {
				onServer = true
			}
		}
		if !p.Combined && p.WebUiDir != "" && serverstate.HasValidServerDeploy(p.WebUiDir) {
			if cfg, err := serverstate.ReadDeployConfig(p.WebUiDir); err == nil {
				publicWebuiURL = serverstate.PublicURL(cfg.StackName)
				if serverstate.OnServer(cfg, 5) {
					onServer = true
				}
			}
		}
	}
	publicStatus := "Down"
	if onServer {
		publicStatus = "UP"
	}

	var skip *string
	if p.SkipReason != "" {
		skip = &p.SkipReason
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
		ApiInternalPort:   p.ApiInternalPort,
		WebUiInternalPort: p.WebUiInternalPort,
		LocalEnabled:      pref.LocalEnabled,
		DockerEnabled:     pref.DockerEnabled,
		PublicEnabled:     pref.PublicEnabled,
		LocalApiURL:       localApiURL,
		LocalWebUiURL:     localWebuiURL,
		LocalStatus:       localStatus,
		DockerApiURL:      dockerApiURL,
		DockerWebUiURL:    dockerWebuiURL,
		DockerStatus:      dockerStatus,
		PublicApiURL:      publicApiURL,
		PublicWebUiURL:    publicWebuiURL,
		PublicStatus:      publicStatus,
		HasServerDeploy:   p.HasServerDeploy,
		OnLocal:           onLocal,
		OnDocker:          onDocker,
		OnServer:          onServer,
		SkipReason:        skip,
		ActionInProgress:  s.router.IsRunning(p.Stem),
	}
}

func (s *Service) UpdateApp(ctx context.Context, stem string, req UpdateRequest) error {
	if req.RunMode == nil || req.Enabled == nil {
		return fmt.Errorf("runMode and enabled are required")
	}
	mode := *req.RunMode
	enabled := *req.Enabled

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
	if mode == runmode.LocalDocker && pair.SkipReason != "" {
		return fmt.Errorf("cannot use docker for %s: %s", stem, pair.SkipReason)
	}
	if mode == runmode.Server && !pair.HasServerDeploy {
		return fmt.Errorf("cannot use public for %s: server deploy scripts missing or invalid", stem)
	}
	if s.router.IsRunning(stem) {
		return fmt.Errorf("action already in progress for %s", stem)
	}

	pref, _, err := s.store.GetAppPreference(ctx, stem)
	if err != nil {
		return err
	}
	switch mode {
	case runmode.Local:
		pref.LocalEnabled = enabled
	case runmode.LocalDocker:
		pref.DockerEnabled = enabled
	case runmode.Server:
		pref.PublicEnabled = enabled
	default:
		return fmt.Errorf("invalid runMode %q", mode)
	}

	if err := s.store.SetAppPreference(ctx, stem, pref); err != nil {
		return err
	}
	if enabled {
		return s.router.Start(ctx, mode, *pair)
	}
	return s.router.Stop(ctx, mode, *pair)
}
