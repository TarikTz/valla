package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestGenerateHTTPClientFile(t *testing.T) {
	out := wiring.GenerateHTTPClientFile("http://localhost:8080")
	if !strings.Contains(out, "http://localhost:8080") {
		t.Error("expected API URL in output")
	}
}
