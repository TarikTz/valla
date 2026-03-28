package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func makeDockerOptions(isSQLite bool, outputMode string) wiring.DockerOptions {
	ctx := registry.WeldContext{
		ProjectName:  "myapp",
		FrontendPort: 5173,
		BackendPort:  8080,
		DBPort:       5432,
		DBUser:       "postgres",
		DBPassword:   "postgres",
		DBName:       "myapp",
		OutputMode:   outputMode,
	}
	frontend := registry.DockerConfig{
		Image:        "node:20-alpine",
		BuildContext: "./frontend",
		Dockerfile:   "registry/dockerfiles/node-vite.Dockerfile",
	}
	backend := registry.DockerConfig{
		Image:        "golang:1.22-alpine",
		BuildContext: "./backend",
		Dockerfile:   "registry/dockerfiles/go.Dockerfile",
	}
	var db *registry.DockerConfig
	if !isSQLite {
		cfg := registry.DockerConfig{Image: "postgres:16-alpine"}
		db = &cfg
	}
	return wiring.DockerOptions{
		Ctx:      ctx,
		Frontend: frontend,
		Backend:  backend,
		DB:       db,
		IsSQLite: isSQLite,
	}
}

func TestGenerateDockerCompose_ContainsServices(t *testing.T) {
	opts := makeDockerOptions(false, "monorepo")
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, serviceName := range []string{"frontend:", "backend:", "db:"} {
		if !strings.Contains(out, serviceName) {
			t.Errorf("expected service %q in compose output", serviceName)
		}
	}
}

func TestGenerateDockerCompose_SQLiteOmitsDB(t *testing.T) {
	opts := makeDockerOptions(true, "monorepo")
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "db:") {
		t.Error("SQLite compose should not contain db service")
	}
}

func TestGenerateDockerCompose_SeparateFolders_BuildContext(t *testing.T) {
	opts := makeDockerOptions(false, "separate")
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "./myapp-frontend") {
		t.Error("separate folders mode should use <project>-frontend as build context")
	}
	if !strings.Contains(out, "./myapp-backend") {
		t.Error("separate folders mode should use <project>-backend as build context")
	}
}
