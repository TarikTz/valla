package registry_test

import (
	"io/fs"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
)

func TestLoad_ReturnsEntries(t *testing.T) {
	entries, err := registry.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one registry entry, got 0")
	}
}

func TestLoad_FindByID(t *testing.T) {
	entries, _ := registry.Load()
	entry, ok := registry.FindByID(entries, "react")
	if !ok {
		t.Fatal("expected to find entry with id=react")
	}
	if entry.Type != "frontend" {
		t.Errorf("expected type=frontend, got %s", entry.Type)
	}
	if entry.DefaultPort != 5173 {
		t.Errorf("expected DefaultPort=5173, got %d", entry.DefaultPort)
	}
}

func TestLoad_FindByType(t *testing.T) {
	entries, _ := registry.Load()
	frontends := registry.FindByType(entries, "frontend")
	if len(frontends) == 0 {
		t.Fatal("expected at least one frontend entry")
	}
}

func TestLoad_DotNetEntries(t *testing.T) {
	entries, err := registry.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	ids := []string{"dotnet-webapi", "dotnet-minimal"}
	for _, id := range ids {
		entry, ok := registry.FindByID(entries, id)
		if !ok {
			t.Errorf("expected to find entry id=%s", id)
			continue
		}
		if entry.Runtime != "dotnet" {
			t.Errorf("%s: expected runtime=dotnet, got %s", id, entry.Runtime)
		}
		if entry.DefaultPort != 5000 {
			t.Errorf("%s: expected port=5000, got %d", id, entry.DefaultPort)
		}
		if entry.CorsPatch == nil {
			t.Errorf("%s: expected cors_patch to be set", id)
		}
		if entry.Docker == nil {
			t.Errorf("%s: expected docker config to be set", id)
		}
		if entry.RequiresRuntime != "dotnet" {
			t.Errorf("%s: expected requires_runtime=dotnet, got %s", id, entry.RequiresRuntime)
		}
	}
}

func TestReadEmbeddedHelpers_NormalizeSourcePaths(t *testing.T) {
	fileBytes, err := registry.ReadEmbeddedFile("internal/registry/data/frontends/react.yaml")
	if err != nil {
		t.Fatalf("ReadEmbeddedFile() error: %v", err)
	}
	if len(fileBytes) == 0 {
		t.Fatal("expected embedded file content")
	}

	dirFS, err := registry.ReadEmbeddedDir("internal/registry/data/templates/backends/node-boilerplate")
	if err != nil {
		t.Fatalf("ReadEmbeddedDir() error: %v", err)
	}
	entries, err := fs.ReadDir(dirFS, ".")
	if err != nil {
		t.Fatalf("ReadDir() error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded directory entries")
	}
}
