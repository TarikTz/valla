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
