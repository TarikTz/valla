package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestGenerateEnv_Postgres_LocalMode(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendPort: 5173,
		BackendPort:  8080,
		DBHost:       "localhost",
		DBPort:       5432,
		DBUser:       "postgres",
		DBPassword:   "postgres",
		DBName:       "myapp",
		EnvMode:      "local",
	}
	out := wiring.GenerateEnv(ctx, false)
	if !strings.Contains(out, "BACKEND_PORT=8080") {
		t.Error("expected BACKEND_PORT=8080")
	}
	if !strings.Contains(out, "DB_HOST=localhost") {
		t.Error("expected DB_HOST=localhost")
	}
	if !strings.Contains(out, "VITE_API_URL=http://localhost:8080") {
		t.Error("expected VITE_API_URL")
	}
}

func TestGenerateEnv_SQLite(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendPort: 5173,
		BackendPort:  8080,
		DBPath:       "./data/app.db",
		EnvMode:      "local",
	}
	out := wiring.GenerateEnv(ctx, true)
	if strings.Contains(out, "DB_HOST") {
		t.Error("SQLite env should not contain DB_HOST")
	}
	if !strings.Contains(out, "DB_PATH=./data/app.db") {
		t.Error("expected DB_PATH")
	}
}

func TestGenerateEnv_DockerMode_DBHost(t *testing.T) {
	ctx := registry.WeldContext{
		BackendPort: 8080,
		DBHost:      "db",
		DBPort:      5432,
		DBUser:      "postgres",
		DBPassword:  "postgres",
		DBName:      "myapp",
		EnvMode:     "docker",
	}
	out := wiring.GenerateEnv(ctx, false)
	if !strings.Contains(out, "DB_HOST=db") {
		t.Error("docker mode should set DB_HOST=db")
	}
}
