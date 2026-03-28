package wiring

import "fmt"

// GenerateHTTPClientFile produces the content of a frontend API client file.
func GenerateHTTPClientFile(apiBaseURL string) string {
	return fmt.Sprintf("export const API_BASE = import.meta.env.VITE_API_URL ?? '%s'\n", apiBaseURL)
}
