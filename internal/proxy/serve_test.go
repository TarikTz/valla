package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"
)

func TestHostnameFromPort(t *testing.T) {
	cases := []struct {
		ns, domain string
		port       int
		want       string
	}{
		{"valla", "test", 5500, "port5500.valla.test"},
		{"my-app", "test", 8080, "port8080.my-app.test"},
		{"valla", "localhost", 3000, "port3000.valla.localhost"},
	}
	for _, tc := range cases {
		got := HostnameFromPort(tc.ns, tc.port, tc.domain)
		if got != tc.want {
			t.Errorf("HostnameFromPort(%q,%d,%q) = %q, want %q", tc.ns, tc.port, tc.domain, got, tc.want)
		}
	}
}

func TestFormatURL(t *testing.T) {
	if got := formatURL("port5500.valla.test", 443); got != "https://port5500.valla.test" {
		t.Errorf("port 443 should omit port number, got %q", got)
	}
	if got := formatURL("port5500.valla.test", 8443); got != "https://port5500.valla.test:8443" {
		t.Errorf("non-443 should include port, got %q", got)
	}
}

func TestBindTLS_FallsBackToOSPort(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCertCache(ca)
	tlsCfg := &tls.Config{GetCertificate: cache.GetCertificate, MinVersion: tls.VersionTLS12}

	// Force OS-assigned port by setting candidates to [0].
	orig := candidatePorts
	candidatePorts = []int{0}
	t.Cleanup(func() { candidatePorts = orig })

	ln, port, err := bindTLS(tlsCfg, false)
	if err != nil {
		t.Fatalf("bindTLS: %v", err)
	}
	defer ln.Close()
	if port <= 0 {
		t.Errorf("expected a positive port, got %d", port)
	}
}

func TestCertCache_GetCertificate_Caches(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCertCache(ca)
	hello := &tls.ClientHelloInfo{ServerName: "port5500.valla.test"}
	c1, err := cache.GetCertificate(hello)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := cache.GetCertificate(hello)
	if err != nil {
		t.Fatal(err)
	}
	if c1 != c2 {
		t.Error("GetCertificate must return the same pointer on subsequent calls (cache hit)")
	}
}

func TestLeafCert_IsSignedByCA(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCertCache(ca)
	host := "api.my-app.test"
	tlsCert, err := cache.Generate([]string{host})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	leaf, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: pool}); err != nil {
		t.Errorf("leaf cert did not verify against CA: %v", err)
	}
}

// TestServe_EndToEnd starts a plain HTTP upstream, wraps it in a TLS proxy,
// and makes an HTTPS request through it using the local CA as trust root.
func TestServe_EndToEnd(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello from upstream")
	}))
	defer upstream.Close()

	var upstreamPort int
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)

	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCertCache(ca)

	// Force OS-assigned proxy port.
	orig := candidatePorts
	candidatePorts = []int{0}
	t.Cleanup(func() { candidatePorts = orig })

	tlsCfg := &tls.Config{GetCertificate: cache.GetCertificate, MinVersion: tls.VersionTLS12}
	ln, proxyPort, err := bindTLS(tlsCfg, false)
	if err != nil {
		t.Fatalf("bindTLS: %v", err)
	}

	hostname := HostnameFromPort("valla", upstreamPort, "test")
	if _, err := cache.Generate([]string{hostname}); err != nil {
		t.Fatalf("pre-warm cert: %v", err)
	}

	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", upstreamPort))
	rp := httputil.NewSingleHostReverseProxy(targetURL)
	srv := &http.Server{Handler: rp}
	go srv.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { srv.Close() })

	// Build a client that trusts our local CA and dials the proxy directly.
	caPool := x509.NewCertPool()
	caPool.AddCert(ca.Cert)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caPool,
				ServerName: hostname,
			},
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
			},
		},
	}

	resp, err := client.Get("https://" + hostname)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello from upstream" {
		t.Errorf("unexpected body: %q", string(body))
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

// TestServe_IntegrationSmoke exercises the full Serve() lifecycle:
// CA creation, TLS binding, proxying a request, and graceful shutdown.
func TestServe_IntegrationSmoke(t *testing.T) {
	// Isolate CA storage in a temp home directory.
	t.Setenv("HOME", t.TempDir())

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "smoke ok")
	}))
	defer upstream.Close()

	var upstreamPort int
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)

	// Force OS-assigned proxy port so we never conflict with a real service.
	orig := candidatePorts
	candidatePorts = []int{0}
	t.Cleanup(func() { candidatePorts = orig })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addrCh := make(chan string, 1)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(ServeOptions{
			TargetPort:   upstreamPort,
			Namespace:    "smoke",
			Domain:       "test",
			Quiet:        true,
			Context:      ctx,
			OnListenAddr: func(addr string) { addrCh <- addr },
		})
	}()

	// Wait for the proxy to be ready.
	select {
	case addr := <-addrCh:
		var proxyPort int
		fmt.Sscanf(addr, "127.0.0.1:%d", &proxyPort)
		if proxyPort <= 0 {
			t.Fatalf("invalid proxy addr %q", addr)
		}

		// Load the freshly-created CA so the test client trusts it.
		ca, _, err := LoadOrCreateCA()
		if err != nil {
			t.Fatalf("LoadOrCreateCA: %v", err)
		}

		hostname := HostnameFromPort("smoke", upstreamPort, "test")
		caPool := x509.NewCertPool()
		caPool.AddCert(ca.Cert)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: caPool, ServerName: hostname},
				DialContext: func(c context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(c, "tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
				},
			},
		}

		resp, err := client.Get("https://" + hostname)
		if err != nil {
			t.Fatalf("GET through proxy: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "smoke ok" {
			t.Errorf("body = %q, want \"smoke ok\"", string(body))
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}

		// Stop the proxy.
		cancel()
		if err := <-serveDone; err != nil {
			t.Errorf("Serve returned error: %v", err)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for proxy to start")
	}
}
