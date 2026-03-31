package registry_test

import (
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
)

func TestWeldContextDevContainerField(t *testing.T) {
	ctx := registry.WeldContext{DevContainer: true}
	if !ctx.DevContainer {
		t.Error("DevContainer field should be settable to true")
	}
}

func TestEntryDevContainerFields(t *testing.T) {
	e := registry.Entry{
		DevContainerImage: "mcr.microsoft.com/devcontainers/go",
		DevCmd:            "go run .",
	}
	if e.DevContainerImage == "" || e.DevCmd == "" {
		t.Error("DevContainerImage and DevCmd fields should be settable")
	}
}
