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
