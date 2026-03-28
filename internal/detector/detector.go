package detector

import "os/exec"

// Detect checks which of the given runtime binaries are available on $PATH.
// Returns a map of binary name → available.
func Detect(runtimes []string) map[string]bool {
	result := make(map[string]bool, len(runtimes))
	for _, rt := range runtimes {
		_, err := exec.LookPath(rt)
		result[rt] = err == nil
	}
	return result
}

// DetectWithAliases returns a map where each key is the logical runtime name
// and the value is true if ANY of the listed alias binaries is found on PATH.
// Map keys are always the logical names (the keys of aliases), never individual binary names.
func DetectWithAliases(aliases map[string][]string) map[string]bool {
	result := make(map[string]bool, len(aliases))
	for logical, binaries := range aliases {
		for _, bin := range binaries {
			if _, err := exec.LookPath(bin); err == nil {
				result[logical] = true
				break
			}
		}
		if !result[logical] {
			result[logical] = false
		}
	}
	return result
}

// FilterByRuntime returns only the runtime names that are present in the available map.
func FilterByRuntime(runtimes []string, available map[string]bool) []string {
	var out []string
	for _, rt := range runtimes {
		if available[rt] {
			out = append(out, rt)
		}
	}
	return out
}
