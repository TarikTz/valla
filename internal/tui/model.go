package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/tui/steps"
)

type phase int

const (
	phaseProjectName phase = iota
	phaseOutputStructure
	phaseFrontendRuntime
	phaseFrontendFramework
	phaseBackendRuntime
	phaseBackendFramework
	phaseDatabaseSelect
	phaseEnvMode
	phasePortOverrides
	phaseConfirm
	phaseExecuting
	phaseDone
)

// Model is the root bubbletea model that sequences all steps.
type Model struct {
	phase   phase
	current tea.Model
	ctx     registry.WeldContext
	entries []registry.Entry

	feRuntimes []string
	beRuntimes []string

	selectedFERuntime string
	selectedBERuntime string
}

// New creates the root model. feRuntimes and beRuntimes are the detected available runtimes.
func New(entries []registry.Entry, feRuntimes, beRuntimes []string) Model {
	return Model{
		phase:      phaseProjectName,
		current:    steps.NewProjectName(),
		entries:    entries,
		feRuntimes: feRuntimes,
		beRuntimes: beRuntimes,
	}
}

func (m Model) Init() tea.Cmd { return m.current.Init() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case steps.StepDone:
		return m.handleStepDone(msg)
	case DoneMsg:
		m.phase = phaseDone
		return m, tea.Quit
	case ErrMsg:
		fmt.Printf("\nError: %v\n", msg.Err)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m Model) handleStepDone(msg steps.StepDone) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseProjectName:
		m.ctx.ProjectName = msg.Value.(string)
		m.ctx.DBName = m.ctx.ProjectName
		m.phase = phaseOutputStructure
		m.current = steps.NewOutputStructure()
	case phaseOutputStructure:
		choice := msg.Value.(string)
		if choice == "WordPress" {
			m.ctx.OutputMode = "wordpress"
			m.ctx.EnvMode = "docker"
			m.ctx.FrontendPort = 8080
			m.ctx.BackendPort = 0
			m.ctx.DBPort = 3306
			m.ctx.DBHost = "db"
			m.ctx.DBUser = "wordpress"
			m.ctx.DBPassword = "wordpress"
			m.ctx.DBName = wordpressDBName(m.ctx.ProjectName)
			m.phase = phasePortOverrides
			m.current = m.buildPortOverrides()
			return m, m.current.Init()
		}
		if len(m.feRuntimes) == 0 || len(m.beRuntimes) == 0 {
			return m, func() tea.Msg {
				return ErrMsg{Err: fmt.Errorf("no supported runtimes detected; install Node.js or Go to continue")}
			}
		}
		if choice == "Monorepo" {
			m.ctx.OutputMode = "monorepo"
		} else {
			m.ctx.OutputMode = "separate"
		}
		m.phase = phaseFrontendRuntime
		m.current = steps.NewRuntimeSelect("Select frontend runtime:", m.feRuntimes)
	case phaseFrontendRuntime:
		m.selectedFERuntime = msg.Value.(string)
		names, _ := m.frameworkDisplayNames("frontend", m.selectedFERuntime)
		m.phase = phaseFrontendFramework
		m.current = steps.NewFrameworkSelect("Select frontend framework:", names)
	case phaseFrontendFramework:
		id := m.frameworkIDFromName("frontend", m.selectedFERuntime, msg.Value.(string))
		m.ctx.FrontendID = id
		entry, _ := registry.FindByID(m.entries, id)
		m.ctx.FrontendPort = entry.DefaultPort
		m.phase = phaseBackendRuntime
		m.current = steps.NewRuntimeSelect("Select backend runtime:", m.beRuntimes)
	case phaseBackendRuntime:
		m.selectedBERuntime = msg.Value.(string)
		names, _ := m.frameworkDisplayNames("backend", m.selectedBERuntime)
		m.phase = phaseBackendFramework
		m.current = steps.NewFrameworkSelect("Select backend framework:", names)
	case phaseBackendFramework:
		id := m.frameworkIDFromName("backend", m.selectedBERuntime, msg.Value.(string))
		m.ctx.BackendID = id
		entry, _ := registry.FindByID(m.entries, id)
		m.ctx.BackendPort = entry.DefaultPort
		m.phase = phaseDatabaseSelect
		m.current = steps.NewDatabaseSelect(m.databaseDisplayNames())
	case phaseDatabaseSelect:
		id := m.databaseIDFromName(msg.Value.(string))
		m.ctx.DatabaseID = id
		entry, _ := registry.FindByID(m.entries, id)
		if entry.SQLite {
			m.ctx.DBPath = entry.DBPathDefault
		} else {
			m.ctx.DBPort = entry.DefaultPort
			m.ctx.DBUser = "postgres"
			m.ctx.DBPassword = "postgres"
		}
		m.phase = phaseEnvMode
		m.current = steps.NewEnvMode()
	case phaseEnvMode:
		choice := msg.Value.(string)
		if choice == "Local (.env)" {
			m.ctx.EnvMode = "local"
			m.ctx.DBHost = "localhost"
		} else {
			m.ctx.EnvMode = "docker"
			m.ctx.DBHost = "db"
		}
		m.phase = phasePortOverrides
		m.current = m.buildPortOverrides()
	case phasePortOverrides:
		result := msg.Value.(steps.PortOverrideResult)
		dbEntry, _ := registry.FindByID(m.entries, m.ctx.DatabaseID)
		m.applyPortOverrides(result, dbEntry.SQLite)
		m.phase = phaseConfirm
		m.current = steps.NewConfirmSummary(m.summaryLines())
	case phaseConfirm:
		return m, tea.Quit
	}
	return m, m.current.Init()
}

// frameworkDisplayNames returns display names for the RuntimeSelect TUI step.
func (m Model) frameworkDisplayNames(entryType, runtime string) (names []string, indexToID map[int]string) {
	indexToID = map[int]string{}
	for _, entry := range m.entries {
		if entry.Type == entryType && entry.Runtime == runtime {
			indexToID[len(names)] = entry.ID
			names = append(names, entry.Name)
		}
	}
	return
}

// frameworkIDFromName resolves a display name back to a registry entry ID.
func (m Model) frameworkIDFromName(entryType, runtime, name string) string {
	for _, entry := range m.entries {
		if entry.Type == entryType && entry.Runtime == runtime && entry.Name == name {
			return entry.ID
		}
	}
	return ""
}

// databaseDisplayNames returns display names for database selection.
func (m Model) databaseDisplayNames() []string {
	var names []string
	for _, entry := range m.entries {
		if entry.Type == "database" {
			names = append(names, entry.Name)
		}
	}
	return names
}

// databaseIDFromName resolves a database display name to a registry entry ID.
func (m Model) databaseIDFromName(name string) string {
	for _, entry := range m.entries {
		if entry.Type == "database" && entry.Name == name {
			return entry.ID
		}
	}
	return ""
}

func (m Model) buildPortOverrides() steps.PortOverrides {
	if m.ctx.OutputMode == "wordpress" {
		return steps.NewPortOverrides([]steps.PortField{
			makePortField("WordPress Port", m.ctx.FrontendPort, false),
			makePortField("MySQL Port", m.ctx.DBPort, false),
		})
	}
	dbEntry, _ := registry.FindByID(m.entries, m.ctx.DatabaseID)
	fields := []steps.PortField{
		makePortField("Frontend Port", m.ctx.FrontendPort, false),
		makePortField("Backend Port", m.ctx.BackendPort, false),
	}
	if dbEntry.SQLite {
		input := textinput.New()
		input.SetValue(m.ctx.DBPath)
		fields = append(fields, steps.PortField{Label: "DB File Path", IsPath: true, Input: input})
	} else {
		fields = append(fields, makePortField("DB Port", m.ctx.DBPort, false))
	}
	return steps.NewPortOverrides(fields)
}

func makePortField(label string, port int, isPath bool) steps.PortField {
	input := textinput.New()
	input.SetValue(fmt.Sprintf("%d", port))
	return steps.PortField{Label: label, IsPath: isPath, Input: input}
}

func (m *Model) applyPortOverrides(result steps.PortOverrideResult, isSQLite bool) {
	parseInt := func(value string, fallback int) int {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		return fallback
	}
	if m.ctx.OutputMode == "wordpress" {
		if len(result.Values) >= 1 {
			m.ctx.FrontendPort = parseInt(result.Values[0], m.ctx.FrontendPort)
		}
		if len(result.Values) >= 2 {
			m.ctx.DBPort = parseInt(result.Values[1], m.ctx.DBPort)
		}
		return
	}
	if len(result.Values) >= 1 {
		m.ctx.FrontendPort = parseInt(result.Values[0], m.ctx.FrontendPort)
	}
	if len(result.Values) >= 2 {
		m.ctx.BackendPort = parseInt(result.Values[1], m.ctx.BackendPort)
	}
	if len(result.Values) >= 3 {
		if isSQLite {
			m.ctx.DBPath = result.Values[2]
		} else {
			m.ctx.DBPort = parseInt(result.Values[2], m.ctx.DBPort)
		}
	}
}

func (m Model) summaryLines() []string {
	if m.ctx.OutputMode == "wordpress" {
		return []string{
			fmt.Sprintf("Project:        %s", m.ctx.ProjectName),
			fmt.Sprintf("Structure:      %s", m.ctx.OutputMode),
			fmt.Sprintf("Env mode:       %s", m.ctx.EnvMode),
			fmt.Sprintf("WordPress port: %d", m.ctx.FrontendPort),
			fmt.Sprintf("MySQL port:     %d", m.ctx.DBPort),
			fmt.Sprintf("DB name:        %s", m.ctx.DBName),
		}
	}
	return []string{
		fmt.Sprintf("Project:    %s", m.ctx.ProjectName),
		fmt.Sprintf("Structure:  %s", m.ctx.OutputMode),
		fmt.Sprintf("Env mode:   %s", m.ctx.EnvMode),
		fmt.Sprintf("FE port:    %d", m.ctx.FrontendPort),
		fmt.Sprintf("BE port:    %d", m.ctx.BackendPort),
	}
}

func (m Model) View() string {
	if m.phase == phaseDone {
		return ""
	}
	return m.current.View()
}

// Context returns the final WeldContext after the TUI completes.
func (m Model) Context() registry.WeldContext { return m.ctx }

func wordpressDBName(projectName string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(projectName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('_')
	}
	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return "wordpress"
	}
	return name
}
