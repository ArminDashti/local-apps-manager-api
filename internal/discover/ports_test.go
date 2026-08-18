package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInternalPortFromYaml(t *testing.T) {
	dir := t.TempDir()
	writeRunYaml(t, dir, `internal_port: "8080"`)
	if got := resolveInternalPort(dir, 80); got != 8080 {
		t.Fatalf("got %d want 8080", got)
	}
}

func TestResolveInternalPortEmptyYamlFallsBackToCompose(t *testing.T) {
	dir := t.TempDir()
	writeRunYaml(t, dir, `internal_port: ""`)
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`
services:
  api:
    expose:
      - "${INTERNAL_PORT:-3000}"
    ports:
      - target: ${INTERNAL_PORT:-3000}
        published: ${PUBLISH_PORT-3000}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveInternalPort(dir, 80); got != 3000 {
		t.Fatalf("got %d want 3000", got)
	}
}

func TestResolveInternalPortSkipsPostgresExpose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`
services:
  postgres:
    expose:
      - "5432"
  api:
    expose:
      - "8080"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveInternalPort(dir, 0); got != 8080 {
		t.Fatalf("got %d want 8080", got)
	}
}

func TestResolveInternalPortFallback(t *testing.T) {
	if got := resolveInternalPort("", 80); got != 80 {
		t.Fatalf("got %d want 80", got)
	}
	if got := resolveInternalPort(t.TempDir(), 0); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func writeRunYaml(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, ".armin", "docker-scripts")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "run-on-docker-local.yaml"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
