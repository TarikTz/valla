package scaffolder_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/scaffolder"
)

func makeDevContainerCtx() registry.WeldContext {
	return registry.WeldContext{
		ProjectName:  "myapp",
		FrontendPort: 3000,
		BackendPort:  8080,
		DevContainer: true,
		OutputMode:   "devcontainer",
		EnvMode:      "docker",
	}
}

func TestGenerateDevContainerFiles_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node"}
	err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir)
	if err != nil {
		t.Fatalf("GenerateDevContainerFiles error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(dir, ".devcontainer", "devcontainer.json"),
		filepath.Join(dir, "docker-compose.dev.yml"),
		filepath.Join(dir, "Makefile"),
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", path)
		}
	}
}

func TestGenerateDevContainerFiles_DevcontainerJSON_ContainsProjectName(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
	if !strings.Contains(string(content), "myapp") {
		t.Error("devcontainer.json should contain project name")
	}
}

func TestGenerateDevContainerFiles_DevcontainerJSON_ForwardsPorts(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
	if !strings.Contains(string(content), "3000") || !strings.Contains(string(content), "8080") {
		t.Error("devcontainer.json should forward frontend and backend ports")
	}
}

func TestGenerateDevContainerFiles_DevcontainerJSON_GoExtension(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go"}
	feEntry := registry.Entry{Runtime: "node"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
	if !strings.Contains(string(content), "golang.go") {
		t.Error("devcontainer.json should include golang.go extension for Go backend")
	}
}

func TestGenerateDevContainerFiles_Compose_ContainsDevCmd(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
	if !strings.Contains(string(content), "go run .") {
		t.Error("docker-compose.dev.yml should contain backend dev command")
	}
}

func TestGenerateDevContainerFiles_Compose_ContainsVolumeMounts(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
	s := string(content)
	if !strings.Contains(s, "./frontend:/app") || !strings.Contains(s, "./backend:/app") {
		t.Error("docker-compose.dev.yml should mount ./frontend:/app and ./backend:/app")
	}
}

func TestGenerateDevContainerFiles_Makefile_ContainsDevTarget(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go"}
	feEntry := registry.Entry{Runtime: "node"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(content), "docker compose") {
		t.Error("Makefile should contain dev target using docker compose")
	}
}

func TestRuntimeToDevContainerImage_Go(t *testing.T) {
	img := scaffolder.RuntimeToDevContainerImage("go", "")
	if img != "mcr.microsoft.com/devcontainers/go" {
		t.Errorf("unexpected image for go: %q", img)
	}
}

func TestRuntimeToDevContainerImage_Override(t *testing.T) {
	img := scaffolder.RuntimeToDevContainerImage("go", "custom-image:latest")
	if img != "custom-image:latest" {
		t.Errorf("expected override image, got %q", img)
	}
}

func TestRuntimeToDevContainerImage_Python3(t *testing.T) {
	img := scaffolder.RuntimeToDevContainerImage("python3", "")
	if img != "mcr.microsoft.com/devcontainers/python" {
		t.Errorf("unexpected image for python3: %q", img)
	}
}

func TestDevCmdForEntry_UsesRegistryValue(t *testing.T) {
	entry := registry.Entry{ID: "go-gin", Runtime: "go", DevCmd: "custom cmd"}
	cmd := scaffolder.DevCmdForEntry(entry)
	if cmd != "custom cmd" {
		t.Errorf("expected entry DevCmd, got %q", cmd)
	}
}

func TestDevCmdForEntry_FallsBackToDefault(t *testing.T) {
	entry := registry.Entry{ID: "go-gin", Runtime: "go"}
	cmd := scaffolder.DevCmdForEntry(entry)
	if cmd != "go run ." {
		t.Errorf("expected default go run ., got %q", cmd)
	}
}

func TestGenerateDevContainerFiles_Node_IsolatesNodeModules(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}
	feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
	s := string(content)
	if !strings.Contains(s, "backend_node_modules:/app/node_modules") {
		t.Error("docker-compose.dev.yml should isolate backend node_modules with named volume")
	}
	if !strings.Contains(s, "frontend_node_modules:/app/node_modules") {
		t.Error("docker-compose.dev.yml should isolate frontend node_modules with named volume")
	}
	// Named volumes must appear in the top-level volumes section.
	if !strings.Contains(s, "backend_node_modules:") {
		t.Error("docker-compose.dev.yml top-level volumes should list backend_node_modules")
	}
	if !strings.Contains(s, "frontend_node_modules:") {
		t.Error("docker-compose.dev.yml top-level volumes should list frontend_node_modules")
	}
}

func TestGenerateDevContainerFiles_Go_NoDepVolume(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
	feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
	s := string(content)
	// Go backend should NOT add a dep volume; node frontend should add one.
	if strings.Contains(s, "backend_gomod") {
		t.Error("go runtime should not add a gomod named volume in the service section")
	}
	if !strings.Contains(s, "frontend_node_modules:/app/node_modules") {
		t.Error("node frontend should still have its dep isolation volume")
	}
}

func TestGenerateDevContainerFiles_Python_IsolatesVenv(t *testing.T) {
	dir := t.TempDir()
	ctx := makeDevContainerCtx()
	beEntry := registry.Entry{Runtime: "python3"}
	feEntry := registry.Entry{Runtime: "node"}
	if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
	s := string(content)
	if !strings.Contains(s, "backend_venv:/app/.venv") {
		t.Error("python3 backend should isolate .venv with named volume")
	}
}
