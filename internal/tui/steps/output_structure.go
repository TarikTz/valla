package steps

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// OutputStructure is the step for choosing a project layout.
// It renders the option list alongside a live directory-tree preview.
type OutputStructure struct {
	options         []string
	cursor          int
	dockerAvailable bool
}

var optionDescriptions = map[string]string{
	"Fully Dockerized": "Packages install inside Docker — your host machine stays untouched. Protects against supply chain attacks.",
	"Monorepo":         "Frontend and backend in one folder with shared config.",
	"Separate folders": "Frontend and backend generated as independent projects.",
	"Frontend only":    "Scaffold the frontend only — no backend generated.",
	"Backend only":     "Scaffold the backend only — no frontend generated.",
	"WordPress":        "WordPress with Docker services and a starter theme.",
}

func NewOutputStructure(dockerAvailable bool) OutputStructure {
	options := []string{"Monorepo", "Separate folders", "Frontend only", "Backend only", "WordPress"}
	if dockerAvailable {
		options = append([]string{"Fully Dockerized"}, options...)
	}
	return OutputStructure{
		options:         options,
		dockerAvailable: dockerAvailable,
	}
}

func (m OutputStructure) Init() tea.Cmd { return nil }

func (m OutputStructure) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m OutputStructure) View() string {
	// Left column: option list
	var left strings.Builder
	left.WriteString(stylePrompt.Render("Project output structure:") + "\n\n")
	for i, option := range m.options {
		if i == m.cursor {
			left.WriteString(styleCursor.Render("›") + " " + styleSelected.Render(option) + "\n")
			if desc, ok := optionDescriptions[option]; ok {
				left.WriteString("    " + styleMuted.Render(desc) + "\n")
			}
		} else {
			left.WriteString("  " + styleOption.Render(option) + "\n")
		}
	}
	left.WriteString("\n" + styleMuted.Render("↑↓ navigate  ·  enter to select  ·  ctrl+c to exit") + "\n")

	leftCol := lipgloss.NewStyle().Width(52).Render(left.String())

	// Right column: live directory tree preview
	preview := structurePreview(m.options[m.cursor])
	rightCol := stylePreviewBox.Render(preview)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

func structurePreview(choice string) string {
	type line struct{ tree, label string }
	tree := func(t, l string) line { return line{t, l} }

	var rows []line
	switch choice {
	case "Monorepo":
		rows = []line{
			tree("", "myapp/"),
			tree("├── ", "frontend/"),
			tree("├── ", "backend/"),
			tree("├── ", ".env"),
			tree("└── ", "docker-compose.yml"),
		}
	case "Separate folders":
		rows = []line{
			tree("", "myapp-frontend/"),
			tree("", "myapp-backend/"),
			tree("", ".env"),
			tree("", "docker-compose.yml"),
		}
	case "Frontend only":
		rows = []line{
			tree("", "myapp/"),
			tree("├── ", "frontend/"),
			tree("└── ", ".env"),
		}
	case "Backend only":
		rows = []line{
			tree("", "myapp/"),
			tree("├── ", "backend/"),
			tree("└── ", ".env"),
		}
	case "WordPress":
		rows = []line{
			tree("", "myapp/"),
			tree("├── ", "wordpress/"),
			tree("│   └── ", "wp-content/"),
			tree("├── ", ".env"),
			tree("└── ", "docker-compose.yml"),
		}
	case "Fully Dockerized":
		rows = []line{
			tree("", "myapp/"),
			tree("├── ", "frontend/"),
			tree("├── ", "backend/"),
			tree("├── ", ".devcontainer/"),
			tree("│   └── ", "devcontainer.json"),
			tree("├── ", ".env"),
			tree("├── ", "docker-compose.yml"),
			tree("├── ", "docker-compose.dev.yml"),
			tree("└── ", "Makefile"),
		}
	default:
		return ""
	}

	var sb strings.Builder
	for _, r := range rows {
		sb.WriteString(stylePreviewTree.Render(r.tree) + stylePreviewDir.Render(r.label) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
