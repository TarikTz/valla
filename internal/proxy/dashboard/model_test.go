package dashboard

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(svcs []Service) Model {
	logCh := make(chan RequestEntry, 10)
	m := New(svcs, "myapp", "test", 8443, logCh)
	m.openURL = func(string) {} // no-op: don't open a real browser
	return m
}

func TestModel_HealthCheckMsg_SetsOnline(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	updated, _ := m.Update(HealthCheckMsg{Index: 0, Online: true})
	got := updated.(Model).services[0]
	if !got.online || !got.checked {
		t.Errorf("expected online=true checked=true, got %+v", got)
	}
}

func TestModel_HealthCheckMsg_SetsDown(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	updated, _ := m.Update(HealthCheckMsg{Index: 0, Online: false})
	got := updated.(Model).services[0]
	if got.online || !got.checked {
		t.Errorf("expected online=false checked=true, got %+v", got)
	}
}

func TestModel_HealthCheckMsg_OutOfBounds(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	// Should not panic for out-of-bounds index.
	_, _ = m.Update(HealthCheckMsg{Index: 99, Online: true})
}

func TestModel_RequestLogMsg_Appends(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	entry := RequestEntry{Method: "GET", Subdomain: "api", Path: "/", StatusCode: 200, Latency: time.Millisecond}
	updated, _ := m.Update(requestLogMsg{Entry: entry})
	logs := updated.(Model).logs
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Path != "/" {
		t.Errorf("unexpected log entry: %+v", logs[0])
	}
}

func TestModel_RequestLogMsg_RollsOver(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	for i := 0; i < 7; i++ {
		e := RequestEntry{Path: "/", StatusCode: 200 + i}
		updated, _ := m.Update(requestLogMsg{Entry: e})
		m = updated.(Model)
	}
	if len(m.logs) > 5 {
		t.Errorf("expected at most 5 log entries, got %d", len(m.logs))
	}
}

func TestModel_KeyQ_Quits(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected a non-nil Cmd for q key")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModel_KeyNumber_CallsOpenURL(t *testing.T) {
	m := newTestModel([]Service{
		{Subdomain: "web", Port: 5500},
		{Subdomain: "api", Port: 8080},
	})
	var opened string
	m.openURL = func(u string) { opened = u }

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if opened != "https://web.myapp.test:8443" {
		t.Errorf("openURL called with %q, want https://web.myapp.test:8443", opened)
	}
}

func TestModel_KeyNumber_OutOfRange(t *testing.T) {
	m := newTestModel([]Service{{Subdomain: "api", Port: 8080}})
	var opened string
	m.openURL = func(u string) { opened = u }
	// '9' is out of range when there is only 1 service
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	if opened != "" {
		t.Errorf("openURL should not be called for out-of-range number, got %q", opened)
	}
}

func TestServiceURL_Port443(t *testing.T) {
	got := serviceURL("api", "myapp", "test", 443)
	want := "https://api.myapp.test"
	if got != want {
		t.Errorf("serviceURL = %q, want %q", got, want)
	}
}

func TestServiceURL_HighPort(t *testing.T) {
	got := serviceURL("api", "myapp", "test", 8443)
	want := "https://api.myapp.test:8443"
	if got != want {
		t.Errorf("serviceURL = %q, want %q", got, want)
	}
}
