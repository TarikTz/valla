package scaffolder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/scaffolder"
)

func TestWritePostScaffoldFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	content := "package main\n\nfunc main() {}\n"
	err := scaffolder.WriteFile(filepath.Join(dir, "main.go"), []byte(content))
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != content {
		t.Errorf("file content mismatch")
	}
}

func TestRenameScaffoldedDir(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "__valla_fe__")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(root, "frontend")
	if err := scaffolder.RenameDir(src, dst); err != nil {
		t.Fatalf("RenameDir error: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Error("expected destination directory to exist")
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("expected source directory to be removed")
	}
}

func TestRollback_DeletesRoot(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "frontend", "src")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "main.go"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := scaffolder.Rollback(root); err != nil {
		t.Fatalf("Rollback error: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Error("expected root directory to be deleted after rollback")
	}
}

func TestApplyTemplate_InterpolatesVariables(t *testing.T) {
	tmplContent := "module {{.ProjectName}}\n\ngo 1.22\n"
	ctx := registry.WeldContext{ProjectName: "myapp"}
	result, err := scaffolder.ApplyTemplate(tmplContent, ctx)
	if err != nil {
		t.Fatalf("ApplyTemplate error: %v", err)
	}
	if result != "module myapp\n\ngo 1.22\n" {
		t.Errorf("unexpected result: %q", result)
	}
}
