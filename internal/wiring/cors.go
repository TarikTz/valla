package wiring

import "strings"

// ApplyCorsPatch finds marker in source and inserts injection on the next line.
// Returns the patched source, whether the marker was found, and an error value.
func ApplyCorsPatch(source, marker, injection string) (string, bool, error) {
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == marker {
			patched := make([]string, 0, len(lines)+1)
			patched = append(patched, lines[:index+1]...)
			patched = append(patched, injection)
			patched = append(patched, lines[index+1:]...)
			return strings.Join(patched, "\n"), true, nil
		}
	}
	return source, false, nil
}
