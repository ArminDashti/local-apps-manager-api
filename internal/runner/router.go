package runner

import (
	"context"
	"fmt"

	"github.com/ArminDashti/local-apps-manager-api/internal/discover"
	"github.com/ArminDashti/local-apps-manager-api/internal/runmode"
)

// ModeRunner starts/stops an app pair for one deployment mode.
type ModeRunner interface {
	Start(ctx context.Context, pair discover.Pair) error
	Stop(ctx context.Context, pair discover.Pair) error
}

// Router dispatches start/stop by run mode and tracks in-flight actions.
type Router struct {
	flight *Flight
	local  ModeRunner
	docker ModeRunner
	server ModeRunner
}

func NewRouter(local, docker, server ModeRunner, flight *Flight) *Router {
	if flight == nil {
		flight = NewFlight()
	}
	return &Router{flight: flight, local: local, docker: docker, server: server}
}

func (r *Router) IsRunning(stem string) bool {
	return r.flight.IsRunning(stem)
}

func (r *Router) Start(ctx context.Context, mode runmode.Mode, pair discover.Pair) error {
	return r.withFlight(pair.Stem, func() error {
		mr, err := r.forMode(mode)
		if err != nil {
			return err
		}
		return mr.Start(ctx, pair)
	})
}

func (r *Router) Stop(ctx context.Context, mode runmode.Mode, pair discover.Pair) error {
	return r.withFlight(pair.Stem, func() error {
		mr, err := r.forMode(mode)
		if err != nil {
			return err
		}
		return mr.Stop(ctx, pair)
	})
}

func (r *Router) withFlight(stem string, fn func() error) error {
	if err := r.flight.acquire(stem); err != nil {
		return err
	}
	defer r.flight.release(stem)
	return fn()
}

func (r *Router) forMode(mode runmode.Mode) (ModeRunner, error) {
	switch mode {
	case runmode.Local:
		return r.local, nil
	case runmode.LocalDocker:
		return r.docker, nil
	case runmode.Server:
		return r.server, nil
	default:
		return nil, fmt.Errorf("unhandled runMode %q", mode)
	}
}
