package wiring

import (
	"fmt"
	"strings"

	"github.com/tariktz/valla-cli/internal/registry"
)

// GenerateEnv produces the content of the .env file.
// isSQLite controls whether to use file-path vars instead of host/port vars.
func GenerateEnv(ctx registry.WeldContext, isSQLite bool) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "FRONTEND_PORT=%d\n", ctx.FrontendPort)
	fmt.Fprintf(&builder, "BACKEND_PORT=%d\n", ctx.BackendPort)
	if isSQLite {
		fmt.Fprintf(&builder, "DB_PATH=%s\n", ctx.DBPath)
	} else {
		fmt.Fprintf(&builder, "DB_HOST=%s\n", ctx.DBHost)
		fmt.Fprintf(&builder, "DB_PORT=%d\n", ctx.DBPort)
		fmt.Fprintf(&builder, "DB_USER=%s\n", ctx.DBUser)
		fmt.Fprintf(&builder, "DB_PASSWORD=%s\n", ctx.DBPassword)
		fmt.Fprintf(&builder, "DB_NAME=%s\n", ctx.DBName)
	}
	fmt.Fprintf(&builder, "VITE_API_URL=http://localhost:%d\n", ctx.BackendPort)
	return builder.String()
}
