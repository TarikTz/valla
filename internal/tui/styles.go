package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleBannerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	styleBannerTagline = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	styleBannerDivider = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#333333"))

	styleUpdateNotice = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	// StyleCardBorder is used by the stack summary card rendered in main.go.
	StyleCardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	// StyleCardLabel is the dim left-side label in a card row.
	StyleCardLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Width(12)

	// StyleCardValue is the bright right-side value in a card row.
	StyleCardValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0"))

	// StyleCardHeader is the "valla · <project>" header line.
	StyleCardHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	// StyleSuccessHeader is the "✓  Done!" line.
	StyleSuccessHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#10B981"))

	// StyleSuccessTree is the muted directory tree.
	StyleSuccessTree = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	// StyleSuccessCmd is the highlighted next-step commands.
	StyleSuccessCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)
)

const bannerASCII = ` 
 ██╗   ██╗  █████╗  ██╗      ██╗       █████╗
 ██║   ██║ ██╔══██╗ ██║      ██║      ██╔══██╗
 ██║   ██║ ███████║ ██║      ██║      ███████║
 ╚██╗ ██╔╝ ██╔══██║ ██║      ██║      ██╔══██║
  ╚████╔╝  ██║  ██║ ███████╗ ███████╗ ██║  ██║
   ╚═══╝   ╚═╝  ╚═╝ ╚══════╝ ╚══════╝ ╚═╝  ╚═╝`

func banner(updateNotice string) string {
	art := styleBannerTitle.Render(bannerASCII)
	tagline := styleBannerTagline.Render("  Scaffold your full stack in seconds.")
	divider := styleBannerDivider.Render("────────────────────────────────────────────────────")
	result := art + "\n" + tagline
	if updateNotice != "" {
		result += "\n" + styleUpdateNotice.Render(updateNotice)
	}
	result += "\n" + divider
	return result
}
