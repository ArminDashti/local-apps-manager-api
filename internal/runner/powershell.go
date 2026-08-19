package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

// Flight tracks in-progress start/stop actions per stem across all mode runners.
type Flight struct {
	mu       sync.Mutex
	inFlight map[string]bool
}

func NewFlight() *Flight {
	return &Flight{inFlight: map[string]bool{}}
}

func (f *Flight) IsRunning(stem string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inFlight[stem]
}

func (f *Flight) acquire(stem string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.inFlight[stem] {
		return fmt.Errorf("action already in progress for %s", stem)
	}
	f.inFlight[stem] = true
	return nil
}

func (f *Flight) release(stem string) {
	f.mu.Lock()
	delete(f.inFlight, stem)
	f.mu.Unlock()
}

func runPowerShell(scriptPath string, extraArgs []string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("runner requires Windows")
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("runner script not found: %w", err)
	}
	args := []string{
		"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
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
