package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type Runner struct {
	scriptPath string
	root       string
	mu         sync.Mutex
	inFlight   map[string]bool
}

func New(scriptPath, root string) *Runner {
	return &Runner{
		scriptPath: scriptPath,
		root:       root,
		inFlight:   map[string]bool{},
	}
}

func (r *Runner) IsRunning(stem string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inFlight[stem]
}

func (r *Runner) Start(stem string) error {
	return r.run(stem, []string{"-Name", stem, "-SkipStopBeforeStart"})
}

func (r *Runner) Stop(stem string) error {
	return r.run(stem, []string{"-StopName", stem})
}

func (r *Runner) run(stem string, extraArgs []string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("docker runner requires Windows")
	}
	r.mu.Lock()
	if r.inFlight[stem] {
		r.mu.Unlock()
		return fmt.Errorf("action already in progress for %s", stem)
	}
	r.inFlight[stem] = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.inFlight, stem)
		r.mu.Unlock()
	}()

	if _, err := os.Stat(r.scriptPath); err != nil {
		return fmt.Errorf("docker runner script not found: %w", err)
	}

	args := []string{
		"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", r.scriptPath,
		"-Root", r.root,
	}
	args = append(args, extraArgs...)

	cmd := exec.Command("powershell.exe", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
