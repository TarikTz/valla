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

	feRuntimeOpts []steps.RuntimeOption
	beRuntimeOpts []steps.RuntimeOption

	selectedFERuntime string
	selectedBERuntime string

	confirmed bool
}

// New creates the root model. feRuntimeOpts and beRuntimeOpts include all known runtimes,
// with Available=false for those not detected on the user's machine.
func New(entries []registry.Entry, feRuntimeOpts, beRuntimeOpts []steps.RuntimeOption) Model {
	return Model{
		phase:         phaseProjectName,
		current:       steps.NewProjectName(),
		entries:       entries,
		feRuntimeOpts: feRuntimeOpts,
		beRuntimeOpts: beRuntimeOpts,
	}
}

// anyAvailable reports whether at least one option in the slice is available.
func anyAvailable(opts []steps.RuntimeOption) bool {
	for _, o := range opts {
		if o.Available {
			return true
		}
	}
	return false
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
			wpDBName := wordpressDBName(m.ctx.ProjectName)
			m.ctx.DatabaseIDs = []string{"mysql"}
			m.ctx.DBConfigs = map[string]registry.DBConfig{
				"mysql": {
					Host:     "db",
					Port:     3306,
					User:     "wordpress",
					Password: "wordpress",
					Name:     wpDBName,
				},
			}
			m.phase = phasePortOverrides
			m.current = m.buildPortOverrides()
			return m, m.current.Init()
		}
		if choice == "Frontend only" {
			if !anyAvailable(m.feRuntimeOpts) {
				return m, func() tea.Msg {
					return ErrMsg{Err: fmt.Errorf("no supported frontend runtimes detected; install Node.js or Bun to continue")}
				}
			}
			m.ctx.OutputMode = "frontend-only"
			m.phase = phaseFrontendRuntime
			m.current = steps.NewRuntimeSelect("Select frontend runtime:", m.feRuntimeOpts)
			return m, m.current.Init()
		}
		if choice == "Backend only" {
			if !anyAvailable(m.beRuntimeOpts) {
				return m, func() tea.Msg {
					return ErrMsg{Err: fmt.Errorf("no supported backend runtimes detected; install Node.js, Go, Python, .NET, or Java (Maven/Gradle) to continue")}
				}
			}
			m.ctx.OutputMode = "backend-only"
			m.phase = phaseBackendRuntime
			m.current = steps.NewRuntimeSelect("Select backend runtime:", m.beRuntimeOpts)
			return m, m.current.Init()
		}
		if !anyAvailable(m.feRuntimeOpts) || !anyAvailable(m.beRuntimeOpts) {
			return m, func() tea.Msg {
				return ErrMsg{Err: fmt.Errorf("no supported runtimes detected; install Node.js, Go, Python, .NET, or Java (Maven/Gradle) to continue")}
			}
		}
		if choice == "Monorepo" {
			m.ctx.OutputMode = "monorepo"
		} else {
			m.ctx.OutputMode = "separate"
		}
		m.phase = phaseFrontendRuntime
		m.current = steps.NewRuntimeSelect("Select frontend runtime:", m.feRuntimeOpts)
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
		if m.ctx.OutputMode == "frontend-only" {
			m.phase = phaseDatabaseSelect
			m.current = m.buildDatabaseMultiSelect()
		} else {
			m.phase = phaseBackendRuntime
			m.current = steps.NewRuntimeSelect("Select backend runtime:", m.beRuntimeOpts)
		}
	case phaseBackendRuntime:
		m.selectedBERuntime = msg.Value.(string)
		grouped := m.frameworkGroupedOptions("backend", m.selectedBERuntime)
		m.phase = phaseBackendFramework
		m.current = steps.NewGroupedRuntimeSelect("Select backend framework:", grouped)
	case phaseBackendFramework:
		id := m.frameworkIDFromName("backend", m.selectedBERuntime, msg.Value.(string))
		m.ctx.BackendID = id
		entry, _ := registry.FindByID(m.entries, id)
		m.ctx.BackendPort = entry.DefaultPort
		m.phase = phaseDatabaseSelect
		m.current = m.buildDatabaseMultiSelect()
	case phaseDatabaseSelect:
		ids := msg.Value.([]string)
		m.ctx.DatabaseIDs = ids
		m.ctx.DBConfigs = map[string]registry.DBConfig{}
		for _, id := range ids {
			entry, _ := registry.FindByID(m.entries, id)
			var cfg registry.DBConfig
			switch id {
			case "postgres":
				cfg = registry.DBConfig{Port: entry.DefaultPort, User: "postgres", Password: "postgres", Name: m.ctx.DBName}
			case "mysql", "mariadb":
				cfg = registry.DBConfig{Port: entry.DefaultPort, User: "root", Password: "root", Name: m.ctx.DBName}
			case "mongodb":
				cfg = registry.DBConfig{Port: entry.DefaultPort, User: "root", Password: "root"}
			case "redis":
				cfg = registry.DBConfig{Port: entry.DefaultPort}
			default:
				if entry.SQLite {
					cfg = registry.DBConfig{Path: entry.DBPathDefault, SQLite: true}
				} else {
					cfg = registry.DBConfig{Port: entry.DefaultPort}
				}
			}
			m.ctx.DBConfigs[id] = cfg
		}
		m.phase = phaseEnvMode
		m.current = steps.NewEnvMode()
	case phaseEnvMode:
		choice := msg.Value.(string)
		var host string
		if choice == "Local (.env)" {
			m.ctx.EnvMode = "local"
			host = "localhost"
		} else {
			m.ctx.EnvMode = "docker"
			host = "db"
		}
		for id, cfg := range m.ctx.DBConfigs {
			cfg.Host = host
			m.ctx.DBConfigs[id] = cfg
		}
		m.phase = phasePortOverrides
		m.current = m.buildPortOverrides()
	case phasePortOverrides:
		result := msg.Value.(steps.PortOverrideResult)
		m.applyPortOverrides(result)
		m.phase = phaseConfirm
		m.current = steps.NewConfirmSummary(m.summaryLines())
	case phaseConfirm:
		m.confirmed = true
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

// frameworkGroupedOptions returns GroupedOptions for use with GroupedRuntimeSelect.
func (m Model) frameworkGroupedOptions(entryType, runtime string) []steps.GroupedOption {
	var opts []steps.GroupedOption
	for _, entry := range m.entries {
		if entry.Type == entryType && entry.Runtime == runtime {
			opts = append(opts, steps.GroupedOption{Name: entry.Name, Group: entry.Group})
		}
	}
	return opts
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

// buildDatabaseMultiSelect builds the multi-select TUI step from database registry entries.
func (m Model) buildDatabaseMultiSelect() steps.DatabaseMultiSelect {
	var opts []steps.DatabaseMultiOption
	for _, entry := range m.entries {
		if entry.Type != "database" {
			continue
		}
		opts = append(opts, steps.DatabaseMultiOption{
			ID:        entry.ID,
			Name:      entry.Name,
			Exclusive: entry.SQLite, // SQLite is exclusive
		})
	}
	return steps.NewDatabaseMultiSelect(opts)
}

func (m Model) buildPortOverrides() steps.PortOverrides {
	if m.ctx.OutputMode == "wordpress" {
		mysqlCfg := m.ctx.DBConfigs["mysql"]
		return steps.NewPortOverrides([]steps.PortField{
			makePortField("WordPress Port", m.ctx.FrontendPort, false),
			makePortField("MySQL Port", mysqlCfg.Port, false),
		})
	}
	var fields []steps.PortField
	if m.ctx.FrontendID != "" {
		fields = append(fields, makePortField("Frontend Port", m.ctx.FrontendPort, false))
	}
	if m.ctx.BackendID != "" {
		fields = append(fields, makePortField("Backend Port", m.ctx.BackendPort, false))
	}
	for _, id := range m.ctx.DatabaseIDs {
		entry, _ := registry.FindByID(m.entries, id)
		cfg := m.ctx.DBConfigs[id]
		if entry.SQLite {
			input := textinput.New()
			input.SetValue(cfg.Path)
			fields = append(fields, steps.PortField{Label: "DB File Path", IsPath: true, Input: input})
		} else {
			fields = append(fields, makePortField(entry.Name+" Port", cfg.Port, false))
		}
	}
	return steps.NewPortOverrides(fields)
}

func makePortField(label string, port int, isPath bool) steps.PortField {
	input := textinput.New()
	input.SetValue(fmt.Sprintf("%d", port))
	return steps.PortField{Label: label, IsPath: isPath, Input: input}
}

func (m *Model) applyPortOverrides(result steps.PortOverrideResult) {
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
			cfg := m.ctx.DBConfigs["mysql"]
			if parsed, err := strconv.Atoi(result.Values[1]); err == nil {
				cfg.Port = parsed
			}
			m.ctx.DBConfigs["mysql"] = cfg
		}
		return
	}
	idx := 0
	if m.ctx.FrontendID != "" && idx < len(result.Values) {
		m.ctx.FrontendPort = parseInt(result.Values[idx], m.ctx.FrontendPort)
		idx++
	}
	if m.ctx.BackendID != "" && idx < len(result.Values) {
		m.ctx.BackendPort = parseInt(result.Values[idx], m.ctx.BackendPort)
		idx++
	}
	for _, id := range m.ctx.DatabaseIDs {
		if idx >= len(result.Values) {
			break
		}
		entry, _ := registry.FindByID(m.entries, id)
		cfg := m.ctx.DBConfigs[id]
		if entry.SQLite {
			cfg.Path = result.Values[idx]
		} else {
			if parsed, err := strconv.Atoi(result.Values[idx]); err == nil {
				cfg.Port = parsed
			}
		}
		m.ctx.DBConfigs[id] = cfg
		idx++
	}
}

func (m Model) summaryLines() []string {
	if m.ctx.OutputMode == "wordpress" {
		mysqlCfg := m.ctx.DBConfigs["mysql"]
		return []string{
			fmt.Sprintf("Project:        %s", m.ctx.ProjectName),
			fmt.Sprintf("Structure:      %s", m.ctx.OutputMode),
			fmt.Sprintf("Env mode:       %s", m.ctx.EnvMode),
			fmt.Sprintf("WordPress port: %d", m.ctx.FrontendPort),
			fmt.Sprintf("MySQL port:     %d", mysqlCfg.Port),
			fmt.Sprintf("DB name:        %s", mysqlCfg.Name),
		}
	}
	lines := []string{
		fmt.Sprintf("Project:    %s", m.ctx.ProjectName),
		fmt.Sprintf("Structure:  %s", m.ctx.OutputMode),
		fmt.Sprintf("Env mode:   %s", m.ctx.EnvMode),
	}
	if m.ctx.FrontendID != "" {
		lines = append(lines, fmt.Sprintf("FE port:    %d", m.ctx.FrontendPort))
	}
	if m.ctx.BackendID != "" {
		lines = append(lines, fmt.Sprintf("BE port:    %d", m.ctx.BackendPort))
	}
	if len(m.ctx.DatabaseIDs) > 0 {
		names := make([]string, 0, len(m.ctx.DatabaseIDs))
		for _, id := range m.ctx.DatabaseIDs {
			entry, _ := registry.FindByID(m.entries, id)
			names = append(names, entry.Name)
		}
		if len(m.ctx.DatabaseIDs) == 1 {
			entry, _ := registry.FindByID(m.entries, m.ctx.DatabaseIDs[0])
			if entry.SQLite {
				lines = append(lines, fmt.Sprintf("DB path:    %s", m.ctx.DBConfigs[m.ctx.DatabaseIDs[0]].Path))
			} else {
				lines = append(lines, fmt.Sprintf("DB port:    %d", m.ctx.DBConfigs[m.ctx.DatabaseIDs[0]].Port))
			}
		} else {
			lines = append(lines, fmt.Sprintf("Databases:  %s", strings.Join(names, ", ")))
		}
	} else {
		lines = append(lines, "Database:   none")
	}
	return lines
}

func (m Model) View() string {
	if m.phase == phaseDone {
		return ""
	}
	return banner() + "\n\n" + m.current.View()
}

// Context returns the final WeldContext after the TUI completes.
func (m Model) Context() registry.WeldContext { return m.ctx }

// Confirmed reports whether the user completed and confirmed the flow.
// Returns false if the user pressed Ctrl+C or cancelled at any point.
func (m Model) Confirmed() bool { return m.confirmed }

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
