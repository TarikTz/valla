package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// candidatePorts is the ordered list of ports the proxy attempts to bind on.
// Overridable in tests.
var candidatePorts = []int{443, 8443, 9443}

// ServeOptions configures a valla serve session.
type ServeOptions struct {
	TargetPort   int             // single-port mode: local port to forward traffic to
	Routes       []Route         // multi-service mode: populated by --map / --range flags
	Namespace    string          // base subdomain namespace (default: "valla")
	Domain       string          // TLD (default: "test")
	Quiet        bool            // suppress non-error output
	Dashboard    bool            // launch the --ui Bubbletea dashboard instead of plain logs
	Expose       bool            // bind to 0.0.0.0 instead of 127.0.0.1 (LAN sharing)
	Context      context.Context // optional parent context; nil uses background
	OnListenAddr func(string)    // called once with "addr:port" when proxy is ready (tests/scripts)
}

// Serve starts the TLS reverse proxy and blocks until stopped (signal or q).
func Serve(opts ServeOptions) error {
	if opts.Namespace == "" {
		opts.Namespace = "valla"
	}
	if opts.Domain == "" {
		opts.Domain = "test"
	}

	if isDotDev(opts.Domain) {
		if !confirmDotDev() {
			return fmt.Errorf("serve: aborted — use --domain test or --domain localhost to avoid HSTS issues")
		}
	}

	if opts.Expose {
		ip, _ := lanIP()
		lanAddr := ip
		if lanAddr == "" {
			lanAddr = "<your-lan-ip>"
		}
		fmt.Fprintf(os.Stderr, "WARNING: --expose binds to 0.0.0.0 — the proxy will be reachable "+
			"by other devices on your local network (LAN address: %s).\n", lanAddr)
	}

	ca, _, err := LoadOrCreateCA()
	if err != nil {
		return fmt.Errorf("loading CA (run 'valla trust' first): %w", err)
	}

	// Resolve routes: multi-service or single-port fallback.
	routes := opts.Routes
	if len(routes) == 0 {
		if opts.TargetPort <= 0 {
			return fmt.Errorf("serve: either a port argument or --map/--range is required")
		}
		routes = []Route{{
			Subdomain: fmt.Sprintf("port%d", opts.TargetPort),
			Port:      opts.TargetPort,
		}}
	}

	table, err := routingTable(opts.Namespace, opts.Domain, routes)
	if err != nil {
		return err
	}

	cache := NewCertCache(ca)
	// Pre-warm certs for every hostname so TLS errors surface before we print URLs.
	for hostname := range table {
		if _, err := cache.Generate([]string{hostname}); err != nil {
			return fmt.Errorf("generating cert for %s: %w", hostname, err)
		}
	}

	handler := buildRoutingHandler(table)
	tlsCfg := &tls.Config{
		GetCertificate: cache.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	ln, proxyPort, err := bindTLS(tlsCfg, opts.Expose)
	if err != nil {
		return err
	}

	if opts.OnListenAddr != nil {
		opts.OnListenAddr(ln.Addr().String())
	}

	if opts.Dashboard {
		return runDashboard(opts, routes, handler, ln, proxyPort)
	}

	if !opts.Quiet {
		fmt.Printf("✓  Valla proxy ready\n\n")
		for _, r := range routes {
			hostname := fmt.Sprintf("%s.%s.%s", r.Subdomain, opts.Namespace, opts.Domain)
			fmt.Printf("  %s  ->  localhost:%d\n", formatURL(hostname, proxyPort), r.Port)
		}
		fmt.Printf("\nPress Ctrl-C or q+Enter to stop.\n\n")
	}

	srv := &http.Server{
		Handler:     handler,
		ReadTimeout: 30 * time.Second,
		// WriteTimeout is intentionally 0 (no limit) because the proxy must
		// support long-lived streaming responses: Vite/webpack HMR event
		// streams, SSE endpoints, and large file downloads would all be
		// killed after a fixed deadline. Slow-loris attacks are not a concern
		// for a loopback-bound dev proxy.
		WriteTimeout: 0,
		// IdleTimeout bounds keep-alive connections so they don't hold
		// resources indefinitely when the upstream goes quiet.
		IdleTimeout: 120 * time.Second,
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigs:
		case <-quitKeyPress():
		}
		cancel()
	}()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// bindTLS tries candidatePorts in order and returns the first successful TLS listener.
// When expose is true it binds to 0.0.0.0 instead of 127.0.0.1.
// Falls back to an OS-assigned port if all candidates fail.
func bindTLS(tlsCfg *tls.Config, expose bool) (net.Listener, int, error) {
	bindHost := "127.0.0.1"
	if expose {
		bindHost = "0.0.0.0"
	}
	for _, port := range candidatePorts {
		addr := fmt.Sprintf("%s:%d", bindHost, port)
		if port == 0 {
			addr = bindHost + ":0"
		}
		ln, err := tls.Listen("tcp", addr, tlsCfg)
		if err == nil {
			actualPort := ln.Addr().(*net.TCPAddr).Port
			return ln, actualPort, nil
		}
	}
	// All candidates failed — let OS pick.
	ln, err := tls.Listen("tcp", bindHost+":0", tlsCfg)
	if err != nil {
		return nil, 0, fmt.Errorf("could not bind on any port: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	return ln, port, nil
}

// HostnameFromPort returns the proxy hostname for a single-port mapping.
// e.g. namespace="valla", port=5500, domain="test" -> "port5500.valla.test"
func HostnameFromPort(namespace string, port int, domain string) string {
	return fmt.Sprintf("port%d.%s.%s", port, namespace, domain)
}

// formatURL returns the HTTPS URL, omitting :port when it is 443.
func formatURL(hostname string, port int) string {
	if port == 443 {
		return "https://" + hostname
	}
	return fmt.Sprintf("https://%s:%d", hostname, port)
}

// isDotDev reports whether domain is "dev" (browser HSTS-preloaded TLD).
func isDotDev(domain string) bool {
	return domain == "dev"
}

// confirmDotDev prints a .dev HSTS warning and prompts the user to confirm.
// Returns true when the user types y/Y, false otherwise.
func confirmDotDev() bool {
	fmt.Fprintln(os.Stderr, "WARNING: .dev is a browser-HSTS-preloaded TLD.")
	fmt.Fprintln(os.Stderr, "         Chrome/Edge enforce HTTPS-only; a misconfigured cert will break your browser.")
	fmt.Fprint(os.Stderr, "Continue anyway? [y/N]: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}

// lanIP returns the first non-loopback IPv4 address of the machine.
func lanIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}
	return "", nil
}

// quitKeyPress returns a channel closed when the user types q/Q followed by Enter.
func quitKeyPress() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 && (buf[0] == 'q' || buf[0] == 'Q') {
				close(ch)
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return ch
}
