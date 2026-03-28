package steps

import tea "github.com/charmbracelet/bubbletea"

// RuntimeSelect is a reusable step for choosing a runtime from a list.
type RuntimeSelect struct {
	prompt  string
	options []string
	cursor  int
}

func NewRuntimeSelect(prompt string, options []string) RuntimeSelect {
	return RuntimeSelect{prompt: prompt, options: options}
}

func (m RuntimeSelect) Init() tea.Cmd { return nil }

func (m RuntimeSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.options) == 0 {
				return m, nil
			}
			return m, func() tea.Msg { return StepDone{Value: m.options[m.cursor]} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m RuntimeSelect) View() string {
	s := stylePrompt.Render(m.prompt) + "\n\n"
	for i, option := range m.options {
		if i == m.cursor {
			s += styleCursor.Render("›") + " " + styleSelected.Render(option) + "\n"
		} else {
			s += "  " + styleOption.Render(option) + "\n"
		}
	}
	return s + "\n" + styleMuted.Render("↑↓ navigate  ·  enter to select") + "\n"
}
