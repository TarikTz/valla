package registry

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data
var registryFS embed.FS

// Load reads all YAML files from the embedded registry directory and returns parsed entries.
func Load() ([]Entry, error) {
	var entries []Entry
	dirs := []string{"data/frontends", "data/backends", "data/databases"}
	for _, dir := range dirs {
		dirEntries, err := fs.ReadDir(registryFS, dir)
		if err != nil {
			// directory may not exist yet (e.g. backends not yet added)
			continue
		}
		for _, de := range dirEntries {
			if de.IsDir() || de.Name()[0] == '.' {
				continue
			}
			data, err := registryFS.ReadFile(dir + "/" + de.Name())
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", de.Name(), err)
			}
			var entry Entry
			if err := yaml.Unmarshal(data, &entry); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", de.Name(), err)
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// FindByID returns the entry with the given ID, if any.
func FindByID(entries []Entry, id string) (Entry, bool) {
	for _, e := range entries {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// FindByType returns all entries of the given type ("frontend", "backend", "database").
func FindByType(entries []Entry, entryType string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Type == entryType {
			out = append(out, e)
		}
	}
	return out
}

// FindByGroup returns all entries belonging to the given group.
func FindByGroup(entries []Entry, group string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Group == group {
			out = append(out, e)
		}
	}
	return out
}

// ReadEmbeddedFile reads a file from the embedded registry FS by path.
// Used by the scaffolder to read post_scaffold_files and builtin_template files.
func ReadEmbeddedFile(path string) ([]byte, error) {
	return registryFS.ReadFile(normalizeEmbeddedPath(path))
}

// ReadEmbeddedDir returns the FS rooted at the given embedded directory.
// Used to copy builtin_template directories.
func ReadEmbeddedDir(path string) (fs.FS, error) {
	return fs.Sub(registryFS, normalizeEmbeddedPath(path))
}

func normalizeEmbeddedPath(path string) string {
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "internal/registry/")
	if strings.HasPrefix(path, "data/") {
		return path
	}
	if strings.HasPrefix(path, "registry/data/") {
		return strings.TrimPrefix(path, "registry/")
	}
	return path
}
