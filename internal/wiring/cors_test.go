package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestApplyCorsPatch_InjectsAfterMarker(t *testing.T) {
	source := "package main\n\nfunc main() {\n// valla:cors\n}\n"
	patched, ok, err := wiring.ApplyCorsPatch(source, "// valla:cors", "app.UseCORS()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected marker to be found")
	}
	if !strings.Contains(patched, "app.UseCORS()") {
		t.Error("expected injected CORS code in output")
	}
	markerIndex := strings.Index(patched, "// valla:cors")
	corsIndex := strings.Index(patched, "app.UseCORS()")
	if corsIndex <= markerIndex {
		t.Error("CORS code should appear after marker")
	}
}

func TestApplyCorsPatch_MarkerNotFound(t *testing.T) {
	source := "package main\n\nfunc main() {}\n"
	_, ok, err := wiring.ApplyCorsPatch(source, "// valla:cors", "app.UseCORS()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false when marker not found")
	}
}
