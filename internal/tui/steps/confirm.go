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
	builder.WriteString(stylePrompt.Render("Summary") + "\n\n")
	for _, line := range m.lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			builder.WriteString("  " + styleLabel.Render(parts[0]+":") + styleSelected.Render(parts[1]) + "\n")
		} else {
			builder.WriteString("  " + styleOption.Render(line) + "\n")
		}
	}
	builder.WriteString("\n")

	var confirm, cancel string
	if m.cursor == 0 {
		confirm = styleConfirmBtn.Render("[ Confirm ]")
		cancel = styleInactiveBtn.Render("[ Cancel ]")
	} else {
		confirm = styleInactiveBtn.Render("[ Confirm ]")
		cancel = styleCancelBtn.Render("[ Cancel ]")
	}
	builder.WriteString(fmt.Sprintf("%s  %s\n", confirm, cancel))
	builder.WriteString("\n" + styleMuted.Render("←→ choose  ·  enter to confirm  ·  ctrl+c to exit") + "\n")
	return builder.String()
}
