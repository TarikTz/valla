package proxy

import (
	"context"
	"net"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tariktz/valla-cli/internal/proxy/dashboard"
)

// runDashboard starts the TLS proxy with Bubbletea UI mode.
// The proxy runs in a background goroutine; the TUI blocks until
// the user presses q or sends SIGINT.
func runDashboard(opts ServeOptions, routes []Route, handler http.Handler, ln net.Listener, proxyPort int) error {
	logCh := make(chan dashboard.RequestEntry, 50)
	loggedHandler := dashboard.WrapHandler(handler, logCh)

	srv := &http.Server{
		Handler:      loggedHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	go srv.Serve(ln) //nolint:errcheck

	svcs := make([]dashboard.Service, len(routes))
	for i, r := range routes {
		svcs[i] = dashboard.Service{Subdomain: r.Subdomain, Port: r.Port}
	}

	dash := dashboard.New(svcs, opts.Namespace, opts.Domain, proxyPort, logCh)
	p := tea.NewProgram(dash, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	return nil
}
