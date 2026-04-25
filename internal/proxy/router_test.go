package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseMap_Valid(t *testing.T) {
	routes, err := ParseMap("ui:3000,api:8080")
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Subdomain != "ui" || routes[0].Port != 3000 {
		t.Errorf("routes[0] = %+v, want {ui 3000}", routes[0])
	}
	if routes[1].Subdomain != "api" || routes[1].Port != 8080 {
		t.Errorf("routes[1] = %+v, want {api 8080}", routes[1])
	}
}

func TestParseMap_Empty(t *testing.T) {
	routes, err := ParseMap("")
	if err != nil {
		t.Fatal(err)
	}
	if routes != nil {
		t.Errorf("empty string should return nil routes, got %v", routes)
	}
}

func TestParseMap_Invalid(t *testing.T) {
	cases := []string{
		"ui",          // no colon
		"ui:notaport", // non-numeric port
		"ui:0",        // port 0 invalid
		":3000",       // empty subdomain
	}
	for _, c := range cases {
		if _, err := ParseMap(c); err == nil {
			t.Errorf("ParseMap(%q): expected error, got nil", c)
		}
	}
}

func TestParseRange_Valid(t *testing.T) {
	routes, err := ParseRange("5500-5502")
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	want := []Route{
		{Subdomain: "port5500", Port: 5500},
		{Subdomain: "port5501", Port: 5501},
		{Subdomain: "port5502", Port: 5502},
	}
	for i, w := range want {
		if routes[i] != w {
			t.Errorf("routes[%d] = %+v, want %+v", i, routes[i], w)
		}
	}
}

func TestParseRange_SinglePort(t *testing.T) {
	routes, err := ParseRange("3000-3000")
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Port != 3000 {
		t.Errorf("unexpected routes: %v", routes)
	}
}

func TestParseRange_Invalid(t *testing.T) {
	cases := []string{
		"5500",      // no dash
		"5502-5500", // end < start
		"abc-5502",  // non-numeric start
		"5500-xyz",  // non-numeric end
	}
	for _, c := range cases {
		if _, err := ParseRange(c); err == nil {
			t.Errorf("ParseRange(%q): expected error, got nil", c)
		}
	}
}

func TestParseMap_And_Range_Combined(t *testing.T) {
	m, err := ParseMap("ui:3000,api:8080")
	if err != nil {
		t.Fatal(err)
	}
	r, err := ParseRange("5500-5501")
	if err != nil {
		t.Fatal(err)
	}
	combined := append(m, r...)
	if len(combined) != 4 {
		t.Errorf("expected 4 combined routes, got %d", len(combined))
	}
}

func TestRoutingHandler_KnownHost(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer upstream.Close()

	var port int
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &port)

	routes := []Route{{Subdomain: "api", Port: port}}
	table, err := routingTable("myapp", "test", routes)
	if err != nil {
		t.Fatal(err)
	}

	handler := buildRoutingHandler(table)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://api.myapp.test/", nil)
	req.Host = "api.myapp.test"
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

func TestRoutingHandler_UnknownHost(t *testing.T) {
	routes := []Route{{Subdomain: "api", Port: 8080}}
	table, err := routingTable("myapp", "test", routes)
	if err != nil {
		t.Fatal(err)
	}
	handler := buildRoutingHandler(table)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://unknown.myapp.test/", nil)
	req.Host = "unknown.myapp.test"
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// TestRoutingHandler_StripsPort verifies that a "host:port" Host header still
// routes correctly to the right upstream.
func TestRoutingHandler_StripsPort(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer upstream.Close()

	var port int
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &port)

	routes := []Route{{Subdomain: "api", Port: port}}
	table, err := routingTable("myapp", "test", routes)
	if err != nil {
		t.Fatal(err)
	}

	handler := buildRoutingHandler(table)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://api.myapp.test/", nil)
	req.Host = "api.myapp.test:8443" // includes port
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after stripping port from Host, got %d", w.Code)
	}
}
