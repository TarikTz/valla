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
)

func banner() string {
	title := styleBannerTitle.Render("⚡ valla-cli")
	tagline := styleBannerTagline.Render("scaffold frontend · backend · database · Docker in one flow")
	divider := styleBannerDivider.Render("────────────────────────────────────────────────────────────")
	return title + "  " + tagline + "\n" + divider
}
