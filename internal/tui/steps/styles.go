package steps

import "github.com/charmbracelet/lipgloss"

var (
	stylePrompt = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	styleCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D7AF"))

	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D7AF"))

	styleOption = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	styleLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	styleMuted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	styleConfirmBtn = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D7AF"))

	styleCancelBtn = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF4444"))

	styleInactiveBtn = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#555555"))

	styleSpinner = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED"))

	styleSpinnerLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#DDDDDD"))
)
