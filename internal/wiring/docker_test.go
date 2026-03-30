package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func baseCtx(outputMode string) registry.WeldContext {
	return registry.WeldContext{
		ProjectName:  "myapp",
		FrontendPort: 5173,
		BackendPort:  8080,
		DBName:       "myapp",
		OutputMode:   outputMode,
	}
}

func makeFrontend() *registry.DockerConfig {
	return &registry.DockerConfig{
		Image:        "node:20-alpine",
		BuildContext: "./frontend",
		Dockerfile:   "registry/dockerfiles/node-vite.Dockerfile",
	}
}

func makeBackend() *registry.DockerConfig {
	return &registry.DockerConfig{
		Image:        "golang:1.22-alpine",
		BuildContext: "./backend",
		Dockerfile:   "registry/dockerfiles/go.Dockerfile",
	}
}

func postgresService() wiring.DBServiceInput {
	return wiring.DBServiceInput{
		ID: "postgres",
		Docker: &registry.DockerConfig{
			Image:      "postgres:16-alpine",
			VolumePath: "/var/lib/postgresql/data",
			EnvVars: map[string]string{
				"POSTGRES_USER":     "{{.User}}",
				"POSTGRES_PASSWORD": "{{.Password}}",
				"POSTGRES_DB":       "{{.Name}}",
			},
		},
		Config: registry.DBConfig{
			Host: "db", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
		},
	}
}

func redisService() wiring.DBServiceInput {
	return wiring.DBServiceInput{
		ID: "redis",
		Docker: &registry.DockerConfig{
			Image:      "redis:7-alpine",
			VolumePath: "/data",
		},
		Config: registry.DBConfig{Host: "db", Port: 6379},
	}
}

func TestGenerateDockerCompose_SingleDB_ContainsServices(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:      baseCtx("monorepo"),
		Frontend: makeFrontend(),
		Backend:  makeBackend(),
		DBs:      []wiring.DBServiceInput{postgresService()},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, svc := range []string{"frontend:", "backend:", "postgres:"} {
		if !strings.Contains(out, svc) {
			t.Errorf("expected service %q in compose output", svc)
		}
	}
	if strings.Contains(out, "db:") {
		t.Error("service should be named 'postgres', not 'db'")
	}
}

func TestGenerateDockerCompose_MultiDB_BothServices(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:     baseCtx("monorepo"),
		Backend: makeBackend(),
		DBs:     []wiring.DBServiceInput{postgresService(), redisService()},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "postgres:") {
		t.Error("expected postgres service")
	}
	if !strings.Contains(out, "redis:") {
		t.Error("expected redis service")
	}
}

func TestGenerateDockerCompose_MultiDB_BothVolumes(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:     baseCtx("monorepo"),
		Backend: makeBackend(),
		DBs:     []wiring.DBServiceInput{postgresService(), redisService()},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "postgres-data:") {
		t.Error("expected postgres-data volume declaration")
	}
	if !strings.Contains(out, "redis-data:") {
		t.Error("expected redis-data volume declaration")
	}
}

func TestGenerateDockerCompose_MultiDB_BackendDependsOnBoth(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:     baseCtx("monorepo"),
		Backend: makeBackend(),
		DBs:     []wiring.DBServiceInput{postgresService(), redisService()},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "postgres") || !strings.Contains(out, "redis") {
		t.Error("backend depends_on should reference both postgres and redis")
	}
}

func TestGenerateDockerCompose_NoDB_NoDBService(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:      baseCtx("monorepo"),
		Frontend: makeFrontend(),
		Backend:  makeBackend(),
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "depends_on") {
		t.Error("no DB: backend should not have depends_on")
	}
}

func TestGenerateDockerCompose_SQLiteOmitsDBService(t *testing.T) {
	sqliteService := wiring.DBServiceInput{
		ID:     "sqlite",
		Docker: nil, // SQLite has no Docker service
		Config: registry.DBConfig{Path: "./data/app.db"},
	}
	opts := wiring.DockerOptions{
		Ctx:     baseCtx("monorepo"),
		Backend: makeBackend(),
		DBs:     []wiring.DBServiceInput{sqliteService},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "sqlite:") {
		t.Error("SQLite should not produce a Docker service")
	}
}

func TestGenerateDockerCompose_SeparateFolders_BuildContext(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:      baseCtx("separate"),
		Frontend: makeFrontend(),
		Backend:  makeBackend(),
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "./myapp-frontend") {
		t.Error("separate mode should use <project>-frontend as build context")
	}
	if !strings.Contains(out, "./myapp-backend") {
		t.Error("separate mode should use <project>-backend as build context")
	}
}

func TestGenerateDockerCompose_PostgresEnvVarsRendered(t *testing.T) {
	opts := wiring.DockerOptions{
		Ctx:     baseCtx("monorepo"),
		Backend: makeBackend(),
		DBs:     []wiring.DBServiceInput{postgresService()},
	}
	out, err := wiring.GenerateDockerCompose(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "POSTGRES_USER: postgres") {
		t.Error("expected rendered POSTGRES_USER: postgres in compose")
	}
	if !strings.Contains(out, "POSTGRES_DB: myapp") {
		t.Error("expected rendered POSTGRES_DB: myapp in compose")
	}
}
