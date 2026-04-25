package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

// Route maps a subdomain label to a local port.
type Route struct {
	Subdomain string
	Port      int
}

// ParseMap parses a comma-separated "subdomain:port" mapping string.
// e.g. "ui:3000,api:8080" → [{Subdomain:"ui",Port:3000},{Subdomain:"api",Port:8080}]
func ParseMap(s string) ([]Route, error) {
	if s == "" {
		return nil, nil
	}
	var routes []Route
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.LastIndex(part, ":")
		if idx < 0 {
			return nil, fmt.Errorf("map entry %q: expected format subdomain:port", part)
		}
		sub := strings.TrimSpace(part[:idx])
		portStr := strings.TrimSpace(part[idx+1:])
		if sub == "" {
			return nil, fmt.Errorf("map entry %q: subdomain must not be empty", part)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("map entry %q: invalid port %q", part, portStr)
		}
		routes = append(routes, Route{Subdomain: sub, Port: port})
	}
	return routes, nil
}

// ParseRange parses a port range (e.g. "5500-5502") into routes with
// auto-generated subdomain names ("port5500", "port5501", "port5502").
func ParseRange(s string) ([]Route, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("range %q: expected format start-end (e.g. 5500-5502)", s)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start <= 0 || start > 65535 {
		return nil, fmt.Errorf("range %q: invalid start port", s)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || end <= 0 || end > 65535 {
		return nil, fmt.Errorf("range %q: invalid end port", s)
	}
	if end < start {
		return nil, fmt.Errorf("range %q: end port must be >= start port", s)
	}
	routes := make([]Route, 0, end-start+1)
	for p := start; p <= end; p++ {
		routes = append(routes, Route{Subdomain: fmt.Sprintf("port%d", p), Port: p})
	}
	return routes, nil
}

// buildRoutingHandler creates an http.Handler that dispatches requests based on the
// Host header. Unknown hostnames return 502 with a human-readable error page.
func buildRoutingHandler(table map[string]*httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// Strip port suffix from Host header (e.g. "api.myapp.test:8443" → "api.myapp.test").
		if i := strings.LastIndex(host, ":"); i >= 0 {
			host = host[:i]
		}
		rp, ok := table[host]
		if !ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "valla: no route for host %q\n\nConfigured routes:\n", r.Host)
			for h := range table {
				fmt.Fprintf(w, "  https://%s\n", h)
			}
			return
		}
		rp.ServeHTTP(w, r)
	})
}

// routingTable constructs a hostname→ReverseProxy map from a list of routes.
func routingTable(namespace, domain string, routes []Route) (map[string]*httputil.ReverseProxy, error) {
	table := make(map[string]*httputil.ReverseProxy, len(routes))
	for _, r := range routes {
		hostname := fmt.Sprintf("%s.%s.%s", r.Subdomain, namespace, domain)
		target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", r.Port))
		if err != nil {
			return nil, err
		}
		table[hostname] = httputil.NewSingleHostReverseProxy(target)
	}
	return table, nil
}
