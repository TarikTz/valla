package steps

import tea "github.com/charmbracelet/bubbletea"

// GroupedOption is an option with a group label for separator rendering.
type GroupedOption struct {
	Name  string
	Group string
}

// GroupedRuntimeSelect is a TUI step that renders options with group separators.
// Separators are purely visual — cursor maps 1:1 to options[i] with no offset.
type GroupedRuntimeSelect struct {
	prompt  string
	options []GroupedOption
	cursor  int
}

// NewGroupedRuntimeSelect creates a GroupedRuntimeSelect step.
func NewGroupedRuntimeSelect(prompt string, options []GroupedOption) GroupedRuntimeSelect {
	return GroupedRuntimeSelect{prompt: prompt, options: options}
}

func (m GroupedRuntimeSelect) Init() tea.Cmd { return nil }

func (m GroupedRuntimeSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, func() tea.Msg { return StepDone{Value: m.options[m.cursor].Name} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m GroupedRuntimeSelect) View() string {
	s := stylePrompt.Render(m.prompt) + "\n\n"
	prevGroup := ""
	multiGroup := hasMultipleGroups(m.options)
	for i, opt := range m.options {
		if opt.Group != prevGroup && prevGroup != "" {
			s += "\n"
		}
		if opt.Group != prevGroup {
			if multiGroup {
				s += styleMuted.Render("── "+opt.Group+" ") + "\n"
			}
			prevGroup = opt.Group
		}
		if i == m.cursor {
			s += styleCursor.Render("›") + " " + styleSelected.Render(opt.Name) + "\n"
		} else {
			s += "  " + styleOption.Render(opt.Name) + "\n"
		}
	}
	return s + "\n" + styleMuted.Render("↑↓ navigate  ·  enter to select  ·  ctrl+c to exit") + "\n"
}

// hasMultipleGroups reports whether the options span more than one group.
func hasMultipleGroups(options []GroupedOption) bool {
	if len(options) == 0 {
		return false
	}
	first := options[0].Group
	for _, o := range options[1:] {
		if o.Group != first {
			return true
		}
	}
	return false
}
