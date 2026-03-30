package tui

import (
	"testing"
)

func TestUpdateAvailableMsgSetsNotice(t *testing.T) {
	m := Model{}
	updatedModel, _ := m.Update(UpdateAvailableMsg{Version: "v9.9.9"})
	got := updatedModel.(Model).updateNotice
	want := "  ⚡ Update available: v9.9.9  →  npm install -g valla-cli"
	if got != want {
		t.Errorf("updateNotice = %q, want %q", got, want)
	}
}

func TestUpdateAvailableMsgEmpty(t *testing.T) {
	m := Model{}
	updatedModel, _ := m.Update(UpdateAvailableMsg{Version: ""})
	got := updatedModel.(Model).updateNotice
	if got != "" {
		t.Errorf("expected empty updateNotice for empty version, got %q", got)
	}
}
