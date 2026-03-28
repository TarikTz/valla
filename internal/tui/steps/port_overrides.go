package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// PortField represents one configurable port or path field.
type PortField struct {
	Label  string
	IsPath bool
	Input  textinput.Model
}

// PortOverrides is the step 9 model: a tabbed form of editable port/path fields.
type PortOverrides struct {
	fields  []PortField
	focused int
}

// PortOverrideResult is the value emitted by StepDone.
type PortOverrideResult struct {
	Values []string
}

func NewPortOverrides(fields []PortField) PortOverrides {
	if len(fields) > 0 {
		fields[0].Input.Focus()
	}
	return PortOverrides{fields: fields, focused: 0}
}

func (m PortOverrides) Init() tea.Cmd { return textinput.Blink }

func (m PortOverrides) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			m.fields[m.focused].Input.Blur()
			m.focused = (m.focused + 1) % len(m.fields)
			m.fields[m.focused].Input.Focus()
		case tea.KeyShiftTab, tea.KeyUp:
			m.fields[m.focused].Input.Blur()
			m.focused = (m.focused - 1 + len(m.fields)) % len(m.fields)
			m.fields[m.focused].Input.Focus()
		case tea.KeyEnter:
			if m.focused == len(m.fields)-1 {
				values := make([]string, len(m.fields))
				for i, field := range m.fields {
					values[i] = field.Input.Value()
				}
				return m, func() tea.Msg { return StepDone{Value: PortOverrideResult{Values: values}} }
			}
			m.fields[m.focused].Input.Blur()
			m.focused = (m.focused + 1) % len(m.fields)
			m.fields[m.focused].Input.Focus()
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.fields[m.focused].Input, cmd = m.fields[m.focused].Input.Update(msg)
	return m, cmd
}

func (m PortOverrides) View() string {
	var builder strings.Builder
	builder.WriteString(stylePrompt.Render("Configure ports") + "\n")
	builder.WriteString(styleMuted.Render("tab/↑↓ navigate  ·  enter on last field to confirm  ·  ctrl+c to exit") + "\n\n")
	for i, field := range m.fields {
		var prefix string
		if i == m.focused {
			prefix = styleCursor.Render("›") + " "
		} else {
			prefix = "  "
		}
		label := field.Label
		if field.IsPath {
			label += " (path)"
		} else {
			label += " (port)"
		}
		var styledLabel string
		if i == m.focused {
			styledLabel = styleSelected.Render(fmt.Sprintf("%-22s", label))
		} else {
			styledLabel = styleLabel.Render(fmt.Sprintf("%-22s", label))
		}
		builder.WriteString(fmt.Sprintf("%s%s %s\n", prefix, styledLabel, field.Input.View()))
	}
	return builder.String()
}
