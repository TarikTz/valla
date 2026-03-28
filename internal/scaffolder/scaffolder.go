package scaffolder

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/tariktz/valla-cli/internal/registry"
)

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// RenameDir renames src to dst.
func RenameDir(src, dst string) error {
	return os.Rename(src, dst)
}

// Rollback deletes the entire project root directory.
func Rollback(root string) error {
	return os.RemoveAll(root)
}

// ApplyTemplate renders a Go text/template string with WeldContext as data.
func ApplyTemplate(tmplStr string, ctx registry.WeldContext) (string, error) {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
