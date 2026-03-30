package wiring

import (
	"fmt"
	"strings"

	"github.com/tariktz/valla-cli/internal/registry"
)

// dbNameVarSuffix returns the env var suffix used for the database/schema name
// for a given registry ID. Returns "" if the DB type doesn't use a name var.
func dbNameVarSuffix(id string) string {
	switch id {
	case "postgres":
		return "DB"
	case "mysql", "mariadb":
		return "DATABASE"
	default:
		return ""
	}
}

// GenerateEnv produces the content of the .env file.
func GenerateEnv(ctx registry.WeldContext) string {
	var builder strings.Builder
	if ctx.FrontendID != "" {
		fmt.Fprintf(&builder, "FRONTEND_PORT=%d\n", ctx.FrontendPort)
	}
	if ctx.BackendID != "" {
		fmt.Fprintf(&builder, "BACKEND_PORT=%d\n", ctx.BackendPort)
	}
	for _, id := range ctx.DatabaseIDs {
		cfg := ctx.DBConfigs[id]
		if cfg.SQLite {
			// SQLite: file-based, no host/port
			fmt.Fprintf(&builder, "DB_PATH=%s\n", cfg.Path)
			continue
		}
		p := strings.ToUpper(id) + "_"
		fmt.Fprintf(&builder, "%sHOST=%s\n", p, cfg.Host)
		fmt.Fprintf(&builder, "%sPORT=%d\n", p, cfg.Port)
		if cfg.User != "" {
			fmt.Fprintf(&builder, "%sUSER=%s\n", p, cfg.User)
			fmt.Fprintf(&builder, "%sPASSWORD=%s\n", p, cfg.Password)
		}
		if suffix := dbNameVarSuffix(id); suffix != "" {
			fmt.Fprintf(&builder, "%s%s=%s\n", p, suffix, cfg.Name)
		}
	}
	if ctx.BackendID != "" {
		fmt.Fprintf(&builder, "VITE_API_URL=http://localhost:%d\n", ctx.BackendPort)
	}
	return builder.String()
}
