package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	yml := "project: film-portal\ndomain: test\nservices:\n  - name: web\n    port: 5500\n    subdomain: preview\n  - name: api\n    port: 8080\n    subdomain: api\n"
	path := writeFile(t, "valla.yaml", yml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Project != "film-portal" {
		t.Errorf("Project = %q, want film-portal", cfg.Project)
	}
	if cfg.Domain != "test" {
		t.Errorf("Domain = %q, want test", cfg.Domain)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("len(Services) = %d, want 2", len(cfg.Services))
	}
	if cfg.Services[0].Port != 5500 || cfg.Services[0].Subdomain != "preview" {
		t.Errorf("Services[0] = %+v", cfg.Services[0])
	}
	if cfg.Services[1].Port != 8080 || cfg.Services[1].Subdomain != "api" {
		t.Errorf("Services[1] = %+v", cfg.Services[1])
	}
}

func TestLoad_NotExist(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "valla.yaml"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for missing file, got %+v", cfg)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeFile(t, "valla.yaml", "not: valid: yaml: {{{{")
	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_MissingProject(t *testing.T) {
	yml := "domain: test\nservices:\n  - name: web\n    port: 5500\n    subdomain: preview\n"
	path := writeFile(t, "valla.yaml", yml)
	if _, err := Load(path); err == nil {
		t.Error("expected error when project field is missing")
	}
}

func TestLoad_NoServices(t *testing.T) {
	yml := "project: my-app\ndomain: test\nservices: []\n"
	path := writeFile(t, "valla.yaml", yml)
	if _, err := Load(path); err == nil {
		t.Error("expected error when services list is empty")
	}
}

func TestLoad_InvalidServicePort(t *testing.T) {
	yml := "project: my-app\nservices:\n  - name: web\n    port: 0\n    subdomain: www\n"
	path := writeFile(t, "valla.yaml", yml)
	if _, err := Load(path); err == nil {
		t.Error("expected error for port 0")
	}
}

func TestLoad_MissingSubdomain(t *testing.T) {
	yml := "project: my-app\nservices:\n  - name: web\n    port: 3000\n"
	path := writeFile(t, "valla.yaml", yml)
	if _, err := Load(path); err == nil {
		t.Error("expected error when subdomain is missing")
	}
}
