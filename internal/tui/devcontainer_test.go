package tui

import (
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/tui/steps"
)

// noneAvailableOpts returns a slice of RuntimeOptions all marked unavailable.
func noneAvailableOpts(names []string) []steps.RuntimeOption {
	opts := make([]steps.RuntimeOption, len(names))
	for i, n := range names {
		opts[i] = steps.RuntimeOption{Name: n, Available: false, Reason: "not found"}
	}
	return opts
}

func TestNewModel_StoresDockerAvailable(t *testing.T) {
	m := New(nil, nil, nil, true)
	if !m.dockerAvailable {
		t.Error("expected dockerAvailable=true to be stored on Model")
	}
}

func TestNewModel_StoresAllRuntimeOpts(t *testing.T) {
	feOpts := noneAvailableOpts([]string{"node", "bun"})
	beOpts := noneAvailableOpts([]string{"go", "node"})

	m := New(nil, feOpts, beOpts, true)

	for _, o := range m.allFeRuntimeOpts {
		if !o.Available {
			t.Errorf("allFeRuntimeOpts: expected %q to be Available=true", o.Name)
		}
	}
	for _, o := range m.allBeRuntimeOpts {
		if !o.Available {
			t.Errorf("allBeRuntimeOpts: expected %q to be Available=true", o.Name)
		}
	}
}

func TestDevContainerPath_SetsContextFields(t *testing.T) {
	feOpts := noneAvailableOpts([]string{"node"})
	beOpts := noneAvailableOpts([]string{"go"})
	m := New([]registry.Entry{}, feOpts, beOpts, true)

	m.ctx.ProjectName = "myapp"
	m.phase = phaseOutputStructure

	updated, _ := m.handleStepDone(steps.StepDone{Value: "Fully Dockerized"})
	result := updated.(Model)

	if result.ctx.OutputMode != "devcontainer" {
		t.Errorf("OutputMode: got %q, want %q", result.ctx.OutputMode, "devcontainer")
	}
	if !result.ctx.DevContainer {
		t.Error("DevContainer should be true")
	}
	if result.ctx.EnvMode != "docker" {
		t.Errorf("EnvMode: got %q, want %q", result.ctx.EnvMode, "docker")
	}
	if result.phase != phaseFrontendRuntime {
		t.Errorf("phase: got %v, want phaseFrontendRuntime", result.phase)
	}
}

func TestDevContainerPath_RuntimeOptsAreAllAvailable(t *testing.T) {
	feOpts := noneAvailableOpts([]string{"node", "bun"})
	beOpts := noneAvailableOpts([]string{"go", "node"})
	m := New([]registry.Entry{}, feOpts, beOpts, true)
	m.phase = phaseOutputStructure

	updated, _ := m.handleStepDone(steps.StepDone{Value: "Fully Dockerized"})
	result := updated.(Model)

	for _, o := range result.feRuntimeOpts {
		if !o.Available {
			t.Errorf("feRuntimeOpts after DevContainer choice: %q should be Available", o.Name)
		}
	}
}

func TestDevContainerPath_SkipsEnvModeAfterDatabaseSelect(t *testing.T) {
	m := Model{
		ctx: registry.WeldContext{DevContainer: true},
	}
	m.phase = phaseDatabaseSelect

	updated, _ := m.handleStepDone(steps.StepDone{Value: []string{}})
	result := updated.(Model)

	if result.phase == phaseEnvMode {
		t.Error("DevContainer path must not transition to phaseEnvMode")
	}
	if result.phase != phasePortOverrides {
		t.Errorf("expected phasePortOverrides, got phase=%v", result.phase)
	}
}

func TestDevContainerPath_SkipsEnvModeAfterORMSelect(t *testing.T) {
	m := Model{
		ctx: registry.WeldContext{DevContainer: true},
	}
	m.phase = phaseORMSelect

	updated, _ := m.handleStepDone(steps.StepDone{Value: "None"})
	result := updated.(Model)

	if result.phase == phaseEnvMode {
		t.Error("DevContainer path must not transition to phaseEnvMode")
	}
	if result.phase != phasePortOverrides {
		t.Errorf("expected phasePortOverrides, got phase=%v", result.phase)
	}
}
