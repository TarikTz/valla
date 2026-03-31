package steps_test

import (
	"testing"

	"github.com/tariktz/valla-cli/internal/tui/steps"
)

func TestAllRuntimeOptions_SetsAllAvailable(t *testing.T) {
	input := []steps.RuntimeOption{
		{Name: "go", Available: false, Reason: "go not found"},
		{Name: "node", Available: true},
		{Name: "python3", Available: false, Reason: "python3 not found"},
	}

	result := steps.AllRuntimeOptions(input)

	for _, opt := range result {
		if !opt.Available {
			t.Errorf("option %q should be Available=true, got false", opt.Name)
		}
	}
}

func TestAllRuntimeOptions_PreservesNamesAndReasons(t *testing.T) {
	input := []steps.RuntimeOption{
		{Name: "go", Available: false, Reason: "go not found"},
	}

	result := steps.AllRuntimeOptions(input)

	if result[0].Name != "go" {
		t.Errorf("expected Name=go, got %q", result[0].Name)
	}
	if result[0].Reason != "go not found" {
		t.Errorf("expected Reason preserved, got %q", result[0].Reason)
	}
}

func TestAllRuntimeOptions_DoesNotMutateInput(t *testing.T) {
	input := []steps.RuntimeOption{
		{Name: "go", Available: false},
	}

	steps.AllRuntimeOptions(input)

	if input[0].Available {
		t.Error("AllRuntimeOptions must not mutate the input slice")
	}
}

func TestAllRuntimeOptions_EmptySlice(t *testing.T) {
	result := steps.AllRuntimeOptions(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got len=%d", len(result))
	}
}
