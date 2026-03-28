package steps

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmSummary shows a summary of all choices and asks for confirmation.
type ConfirmSummary struct {
	lines  []string
	cursor int
}

func NewConfirmSummary(lines []string) ConfirmSummary {
	return ConfirmSummary{lines: lines}
}

func (m ConfirmSummary) Init() tea.Cmd { return nil }

func (m ConfirmSummary) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.cursor = 0
		case "right", "l":
			m.cursor = 1
		case "enter":
			if m.cursor == 0 {
				return m, func() tea.Msg { return StepDone{Value: true} }
			}
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ConfirmSummary) View() string {
	var builder strings.Builder
	builder.WriteString("Summary:\n\n")
	for _, line := range m.lines {
		builder.WriteString("  " + line + "\n")
	}
	builder.WriteString("\n")
	confirm := "[ Confirm ]"
	cancel := "[ Cancel ]"
	if m.cursor == 0 {
		confirm = "[>Confirm<]"
	} else {
		cancel = "[>Cancel<]"
	}
	builder.WriteString(fmt.Sprintf("%s  %s\n", confirm, cancel))
	return builder.String()
}
