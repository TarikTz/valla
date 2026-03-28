package steps_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tariktz/valla-cli/internal/tui/steps"
)

func makeField(label, value string) steps.PortField {
	ti := textinput.New()
	ti.SetValue(value)
	return steps.PortField{Label: label, IsPath: false, Input: ti}
}

func TestPortOverrides_TabAdvancesFocus(t *testing.T) {
	model := steps.NewPortOverrides([]steps.PortField{
		makeField("Frontend Port", "5173"),
		makeField("Backend Port", "8080"),
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	portOverrides := updated.(steps.PortOverrides)
	_ = portOverrides.View()
}

func TestPortOverrides_EnterOnLastFieldEmitsStepDone(t *testing.T) {
	model := steps.NewPortOverrides([]steps.PortField{
		makeField("Frontend Port", "5173"),
		makeField("Backend Port", "8080"),
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command to be returned on Enter on last field")
	}

	msg := cmd()
	done, ok := msg.(steps.StepDone)
	if !ok {
		t.Fatalf("expected StepDone, got %T", msg)
	}

	result, ok := done.Value.(steps.PortOverrideResult)
	if !ok {
		t.Fatalf("expected PortOverrideResult, got %T", done.Value)
	}
	if len(result.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(result.Values))
	}
	if result.Values[0] != "5173" {
		t.Errorf("expected first value 5173, got %s", result.Values[0])
	}
}
