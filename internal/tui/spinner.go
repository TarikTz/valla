package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerMsg signals an update to the spinner label.
type SpinnerMsg struct{ Label string }

// DoneMsg signals all scaffolding is complete.
type DoneMsg struct{ NextSteps string }

// ErrMsg signals a fatal error during scaffolding.
type ErrMsg struct{ Err error }

// Spinner is the execution-phase model shown after user confirms.
type Spinner struct {
	sp    spinner.Model
	label string
}

func NewSpinner() Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Spinner{sp: s, label: "Initializing..."}
}

func (m Spinner) Init() tea.Cmd { return m.sp.Tick }

func (m Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SpinnerMsg:
		m.label = msg.Label
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Spinner) View() string {
	return m.sp.View() + " " + m.label + "\n"
}
