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

// sqlDatabases is the set of SQL database registry IDs eligible for DATABASE_URL.
var sqlDatabases = map[string]bool{
	"postgres": true,
	"mysql":    true,
	"mariadb":  true,
	"sqlite":   true,
}

// composeDatabaseURL returns a DATABASE_URL connection string for the given DB ID and config.
// Returns "" for non-SQL databases (redis, mongodb).
func composeDatabaseURL(id string, cfg registry.DBConfig) string {
	switch id {
	case "postgres":
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	case "mysql", "mariadb":
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	case "sqlite":
		return "file:" + cfg.Path
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
	// Emit DATABASE_URL for the primary SQL database when an ORM is selected.
	if ctx.ORMID != "" {
		for _, id := range ctx.DatabaseIDs {
			if sqlDatabases[id] {
				if url := composeDatabaseURL(id, ctx.DBConfigs[id]); url != "" {
					fmt.Fprintf(&builder, "DATABASE_URL=%s\n", url)
				}
				break // only the first SQL DB gets DATABASE_URL
			}
		}
	}
	if ctx.BackendID != "" {
		fmt.Fprintf(&builder, "VITE_API_URL=http://localhost:%d\n", ctx.BackendPort)
	}
	return builder.String()
}
