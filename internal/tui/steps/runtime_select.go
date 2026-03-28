package steps

import tea "github.com/charmbracelet/bubbletea"

// RuntimeOption represents one selectable runtime.
// Disabled entries are shown but cannot be navigated to or selected.
type RuntimeOption struct {
	Name      string
	Available bool
	Reason    string // displayed when Available is false
}

// RuntimeSelect is a reusable step for choosing a runtime from a list.
type RuntimeSelect struct {
	prompt  string
	options []RuntimeOption
	cursor  int
}

// runtimeOptionsAll wraps a plain string slice as available RuntimeOptions.
// Used by components that reuse RuntimeSelect for non-runtime selection
// (framework, database, env-mode) where all options are always selectable.
func runtimeOptionsAll(names []string) []RuntimeOption {
	opts := make([]RuntimeOption, len(names))
	for i, n := range names {
		opts[i] = RuntimeOption{Name: n, Available: true}
	}
	return opts
}

func NewRuntimeSelect(prompt string, options []RuntimeOption) RuntimeSelect {
	// Initialise cursor on the first available option.
	cursor := 0
	for i, o := range options {
		if o.Available {
			cursor = i
			break
		}
	}
	return RuntimeSelect{prompt: prompt, options: options, cursor: cursor}
}

func (m RuntimeSelect) Init() tea.Cmd { return nil }

func (m RuntimeSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			prev := m.cursor - 1
			for prev >= 0 && !m.options[prev].Available {
				prev--
			}
			if prev >= 0 {
				m.cursor = prev
			}
		case "down", "j":
			next := m.cursor + 1
			for next < len(m.options) && !m.options[next].Available {
				next++
			}
			if next < len(m.options) {
				m.cursor = next
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

func (m RuntimeSelect) View() string {
	s := stylePrompt.Render(m.prompt) + "\n\n"
	for i, opt := range m.options {
		if !opt.Available {
			s += "  " + styleMuted.Render(opt.Name+"  — "+opt.Reason) + "\n"
			continue
		}
		if i == m.cursor {
			s += styleCursor.Render("›") + " " + styleSelected.Render(opt.Name) + "\n"
		} else {
			s += "  " + styleOption.Render(opt.Name) + "\n"
		}
	}
	return s + "\n" + styleMuted.Render("↑↓ navigate  ·  enter to select  ·  ctrl+c to exit") + "\n"
}
