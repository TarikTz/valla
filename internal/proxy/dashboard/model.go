package dashboard

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Service describes a single proxied upstream for the dashboard.
type Service struct {
	Subdomain string
	Port      int
}

// RequestEntry is one HTTP request captured by WrapHandler.
type RequestEntry struct {
	Method     string
	Subdomain  string
	Path       string
	StatusCode int
	Latency    time.Duration
}

// HealthCheckMsg is sent by the background health-check goroutine.
type HealthCheckMsg struct {
	Index  int
	Online bool
}

// requestLogMsg wraps RequestEntry as a Bubbletea message.
type requestLogMsg struct {
	Entry RequestEntry
}

// healthTickMsg triggers a new round of health checks.
type healthTickMsg struct{}

// serviceState holds runtime status for one service entry.
type serviceState struct {
	Service
	online  bool
	checked bool // true after at least one health check
}

// Model is the Bubbletea model for the valla --ui dashboard.
type Model struct {
	services  []serviceState
	logs      []RequestEntry // rolling last 5
	namespace string
	domain    string
	proxyPort int
	logCh     <-chan RequestEntry
	openURL   func(string) // injectable; defaults to openBrowser
}

var (
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	styleOnline = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	styleDown   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	styleURL    = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

// New creates a dashboard Model for the given routes.
func New(services []Service, namespace, domain string, proxyPort int, logCh <-chan RequestEntry) Model {
	states := make([]serviceState, len(services))
	for i, s := range services {
		states[i] = serviceState{Service: s}
	}
	return Model{
		services:  states,
		namespace: namespace,
		domain:    domain,
		proxyPort: proxyPort,
		logCh:     logCh,
		openURL:   openBrowser,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.services)+2)
	for i, s := range m.services {
		cmds = append(cmds, checkHealth(i, s.Port))
	}
	cmds = append(cmds, waitForLog(m.logCh))
	cmds = append(cmds, tea.Tick(4*time.Second, func(time.Time) tea.Msg { return healthTickMsg{} }))
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HealthCheckMsg:
		if msg.Index >= 0 && msg.Index < len(m.services) {
			m.services[msg.Index].online = msg.Online
			m.services[msg.Index].checked = true
		}
		return m, nil

	case requestLogMsg:
		m.logs = append(m.logs, msg.Entry)
		if len(m.logs) > 5 {
			m.logs = m.logs[len(m.logs)-5:]
		}
		return m, waitForLog(m.logCh)

	case healthTickMsg:
		cmds := make([]tea.Cmd, 0, len(m.services)+1)
		for i, s := range m.services {
			cmds = append(cmds, checkHealth(i, s.Port))
		}
		cmds = append(cmds, tea.Tick(4*time.Second, func(time.Time) tea.Msg { return healthTickMsg{} }))
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		default:
			// number keys 1-9: open corresponding service in browser
			if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
				idx := int(msg.Runes[0]-'1')
				if idx < len(m.services) {
					u := serviceURL(m.services[idx].Subdomain, m.namespace, m.domain, m.proxyPort)
					m.openURL(u)
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	const divider = "────────────────────────────────────────────────────────────────────────"
	var sb strings.Builder

	title := fmt.Sprintf("Valla Proxy Active  |  %s  |  .%s", m.namespace, m.domain)
	sb.WriteString("\n  " + styleTitle.Render(title) + "\n")
	sb.WriteString("  " + styleDim.Render(divider) + "\n\n")

	// Header row
	sb.WriteString(fmt.Sprintf("  %-3s  %-10s  %-12s  %-18s  %s\n",
		"#", "STATUS", "SUBDOMAIN", "TARGET", "HTTPS URL"))
	sb.WriteString("  " + styleDim.Render(divider) + "\n")

	for i, svc := range m.services {
		var statusStr string
		switch {
		case !svc.checked:
			statusStr = styleDim.Render("○ WAIT   ")
		case svc.online:
			statusStr = styleOnline.Render("● ONLINE ")
		default:
			statusStr = styleDown.Render("○ DOWN   ")
		}
		target := fmt.Sprintf("localhost:%d", svc.Port)
		u := serviceURL(svc.Subdomain, m.namespace, m.domain, m.proxyPort)
		sb.WriteString(fmt.Sprintf("  %-3d  %-10s  %-12s  %-18s  %s\n",
			i+1, statusStr, svc.Subdomain, target, styleURL.Render(u)))
	}

	sb.WriteString("  " + styleDim.Render(divider) + "\n")
	sb.WriteString("  " + styleDim.Render("[Recent requests]") + "\n")
	if len(m.logs) == 0 {
		sb.WriteString("  " + styleDim.Render("—") + "\n")
	}
	for _, l := range m.logs {
		sb.WriteString(fmt.Sprintf("  %-5s  %-12s  %-24s  %d  %s\n",
			l.Method, l.Subdomain, l.Path, l.StatusCode,
			l.Latency.Round(time.Millisecond)))
	}
	sb.WriteString("  " + styleDim.Render(divider) + "\n")
	sb.WriteString("  " + styleDim.Render("Press 1-9 to open in browser · q to quit") + "\n")

	return sb.String()
}

// serviceURL builds an HTTPS URL for a given service, omitting :443.
func serviceURL(subdomain, namespace, domain string, proxyPort int) string {
	hostname := fmt.Sprintf("%s.%s.%s", subdomain, namespace, domain)
	if proxyPort == 443 {
		return "https://" + hostname
	}
	return fmt.Sprintf("https://%s:%d", hostname, proxyPort)
}

// checkHealth runs a GET to the upstream port and sends back a HealthCheckMsg.
func checkHealth(index, port int) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 2 * time.Second}
		resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			_ = resp.Body.Close()
		}
		return HealthCheckMsg{Index: index, Online: err == nil}
	}
}

// waitForLog returns a Cmd that blocks until the next RequestEntry is ready.
// Returns nil when the channel is closed (proxy shutting down), which stops
// the Bubbletea command loop for logging without leaking a goroutine.
func waitForLog(ch <-chan RequestEntry) tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-ch
		if !ok {
			return nil
		}
		return requestLogMsg{Entry: entry}
	}
}

// openBrowser opens url in the default OS browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		// Use rundll32 instead of "cmd /c start" to avoid shell interpretation
		// of characters like & | ^ that could appear in a crafted subdomain.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
