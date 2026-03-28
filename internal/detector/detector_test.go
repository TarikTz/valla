package detector_test

import (
	"testing"

	"github.com/tariktz/valla-cli/internal/detector"
)

func TestDetect_ReturnsMap(t *testing.T) {
	runtimes := detector.Detect([]string{"go", "node", "python3", "dotnet"})
	if runtimes == nil {
		t.Fatal("Detect() returned nil map")
	}
	if !runtimes["go"] {
		t.Log("go not detected — expected on dev machine")
	}
}

func TestDetect_MissingRuntime(t *testing.T) {
	runtimes := detector.Detect([]string{"__nonexistent_binary_xyz__"})
	if runtimes["__nonexistent_binary_xyz__"] {
		t.Error("expected nonexistent binary to return false")
	}
}

func TestDetectWithAliases_ReturnsTrueWhenAnyAliasFound(t *testing.T) {
	// "go" binary is available on any dev machine building this project
	result := detector.DetectWithAliases(map[string][]string{
		"java": {"__nonexistent_xyz__", "go"},
	})
	if !result["java"] {
		t.Error("expected java=true when at least one alias (go) is found")
	}
}

func TestDetectWithAliases_ReturnsFalseWhenNoAliasFound(t *testing.T) {
	result := detector.DetectWithAliases(map[string][]string{
		"java": {"__nonexistent_xyz__", "__also_missing__"},
	})
	if result["java"] {
		t.Error("expected java=false when no alias is found")
	}
}

func TestDetectWithAliases_KeyIsLogicalName(t *testing.T) {
	result := detector.DetectWithAliases(map[string][]string{
		"myjava": {"go"},
	})
	if _, ok := result["myjava"]; !ok {
		t.Error("expected map key to be logical name 'myjava', not the alias binary name")
	}
	if result["go"] {
		t.Error("expected binary alias name 'go' NOT to be a key in the result")
	}
}

func TestFilterEntries_RemovesUnavailable(t *testing.T) {
	runtimes := map[string]bool{"go": true}
	available := detector.FilterByRuntime(
		[]string{"go", "node"},
		runtimes,
	)
	if len(available) != 1 || available[0] != "go" {
		t.Errorf("expected [go], got %v", available)
	}
}
