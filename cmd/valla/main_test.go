package main

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
)

func TestRenderSummaryCardFullStack(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName: "my-app",
		FrontendID:  "react-node",
		BackendID:   "gin",
		DatabaseIDs: []string{"postgres"},
		ORMID:       "drizzle",
		OutputMode:  "monorepo",
		EnvMode:     "docker",
	}
	entries := []registry.Entry{
		{ID: "react-node", Name: "React", Runtime: "node", Type: "frontend"},
		{ID: "gin", Name: "Gin", Runtime: "go", Type: "backend"},
		{ID: "postgres", Name: "PostgreSQL", Type: "database"},
	}
	card := renderSummaryCard(ctx, entries)
	for _, want := range []string{"my-app", "React", "Gin", "PostgreSQL", "Drizzle", "Monorepo", "Docker"} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q\ncard:\n%s", want, card)
		}
	}
}

func TestRenderSummaryCardWordPress(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName:  "my-wp",
		OutputMode:   "wordpress",
		EnvMode:      "docker",
		FrontendPort: 8080,
		DatabaseIDs:  []string{"mysql"},
		DBConfigs:    map[string]registry.DBConfig{"mysql": {Port: 3306}},
	}
	card := renderSummaryCard(ctx, nil)
	if !strings.Contains(card, "my-wp") {
		t.Errorf("card missing project name\ncard:\n%s", card)
	}
	if strings.Contains(card, "Frontend") || strings.Contains(card, "Backend") {
		t.Errorf("wordpress card should not have Frontend/Backend rows\ncard:\n%s", card)
	}
}

func TestRenderSuccessOutputMonorepoDocker(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName: "my-app",
		FrontendID:  "react-node",
		BackendID:   "gin",
		OutputMode:  "monorepo",
		EnvMode:     "docker",
	}
	feEntry := registry.Entry{ID: "react-node", Name: "React", Runtime: "node"}
	beEntry := registry.Entry{ID: "gin", Name: "Gin", Runtime: "go"}
	out := renderSuccessOutput(ctx, "frontend", "backend", feEntry, beEntry, "")
	for _, want := range []string{"Done", "my-app", "docker"} {
		if !strings.Contains(out, want) {
			t.Errorf("success output missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestParseLatestVersion(t *testing.T) {
	body := `{"tag_name":"v1.2.3","name":"Release v1.2.3"}`
	got := parseTagFromBody([]byte(body))
	if got != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", got)
	}
}

func TestParseLatestVersionMissing(t *testing.T) {
	got := parseTagFromBody([]byte(`{}`))
	if got != "" {
		t.Errorf("expected empty string for missing tag_name, got %q", got)
	}
}
