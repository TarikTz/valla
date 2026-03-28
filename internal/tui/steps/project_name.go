package steps

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ProjectName is the bubbletea model for step 1: entering the project name.
type ProjectName struct {
	input textinput.Model
}

func NewProjectName() ProjectName {
	ti := textinput.New()
	ti.Placeholder = "my-app"
	ti.Focus()
	return ProjectName{input: ti}
}

func (m ProjectName) Init() tea.Cmd { return textinput.Blink }

func (m ProjectName) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.input.Value() != "" {
				return m, func() tea.Msg { return StepDone{Value: m.input.Value()} }
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ProjectName) View() string {
	label := stylePrompt.Render("Project name")
	hint := styleMuted.Render("  ·  enter a name for your new project")
	return label + hint + "\n\n" + m.input.View() + "\n\n" + styleMuted.Render("enter to continue") + "\n"
}
