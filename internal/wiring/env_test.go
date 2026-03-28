package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestGenerateEnv_Postgres_LocalMode(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendID:   "react",
		BackendID:    "go-gin",
		DatabaseID:   "postgres",
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
		FrontendID:   "react",
		BackendID:    "go-gin",
		DatabaseID:   "sqlite",
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
		BackendID:  "go-gin",
		DatabaseID: "postgres",
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

func TestGenerateEnv_NoDatabase(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendID:   "react",
		BackendID:    "go-gin",
		FrontendPort: 5173,
		BackendPort:  8080,
		EnvMode:      "local",
	}
	out := wiring.GenerateEnv(ctx, false)
	if strings.Contains(out, "DB_") {
		t.Error("no DB selected — env should not contain DB_ vars")
	}
}

func TestGenerateEnv_FrontendOnly(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendID:   "react",
		FrontendPort: 5173,
		EnvMode:      "local",
	}
	out := wiring.GenerateEnv(ctx, false)
	if !strings.Contains(out, "FRONTEND_PORT=5173") {
		t.Error("expected FRONTEND_PORT")
	}
	if strings.Contains(out, "BACKEND_PORT") {
		t.Error("frontend-only should not contain BACKEND_PORT")
	}
	if strings.Contains(out, "VITE_API_URL") {
		t.Error("frontend-only should not contain VITE_API_URL")
	}
}
