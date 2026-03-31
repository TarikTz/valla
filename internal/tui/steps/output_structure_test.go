package steps_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/tui/steps"
)

func TestOutputStructure_WithDocker_ShowsFullyDockerized(t *testing.T) {
	m := steps.NewOutputStructure(true)
	view := m.View()
	if !strings.Contains(view, "Fully Dockerized") {
		t.Error("expected 'Fully Dockerized' option when dockerAvailable=true")
	}
}

func TestOutputStructure_WithoutDocker_HidesFullyDockerized(t *testing.T) {
	m := steps.NewOutputStructure(false)
	view := m.View()
	if strings.Contains(view, "Fully Dockerized") {
		t.Error("expected 'Fully Dockerized' option to be hidden when dockerAvailable=false")
	}
}
