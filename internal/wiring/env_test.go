package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func makeEnvCtx() registry.WeldContext {
	return registry.WeldContext{
		FrontendID:   "react",
		BackendID:    "go-gin",
		FrontendPort: 5173,
		BackendPort:  8080,
		DBName:       "myapp",
		EnvMode:      "local",
	}
}

func addDB(ctx registry.WeldContext, id string, cfg registry.DBConfig) registry.WeldContext {
	ctx.DatabaseIDs = append(ctx.DatabaseIDs, id)
	if ctx.DBConfigs == nil {
		ctx.DBConfigs = map[string]registry.DBConfig{}
	}
	ctx.DBConfigs[id] = cfg
	return ctx
}

func TestGenerateEnv_Postgres_LocalMode(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "BACKEND_PORT=8080") {
		t.Error("expected BACKEND_PORT=8080")
	}
	if !strings.Contains(out, "POSTGRES_HOST=localhost") {
		t.Error("expected POSTGRES_HOST=localhost")
	}
	if !strings.Contains(out, "POSTGRES_PORT=5432") {
		t.Error("expected POSTGRES_PORT=5432")
	}
	if !strings.Contains(out, "POSTGRES_USER=postgres") {
		t.Error("expected POSTGRES_USER=postgres")
	}
	if !strings.Contains(out, "POSTGRES_PASSWORD=postgres") {
		t.Error("expected POSTGRES_PASSWORD=postgres")
	}
	if !strings.Contains(out, "POSTGRES_DB=myapp") {
		t.Error("expected POSTGRES_DB=myapp")
	}
	if !strings.Contains(out, "VITE_API_URL=http://localhost:8080") {
		t.Error("expected VITE_API_URL")
	}
}

func TestGenerateEnv_MySQL(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "mysql", registry.DBConfig{
		Host: "localhost", Port: 3306, User: "root", Password: "root", Name: "myapp",
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "MYSQL_HOST=localhost") {
		t.Error("expected MYSQL_HOST=localhost")
	}
	if !strings.Contains(out, "MYSQL_DATABASE=myapp") {
		t.Error("expected MYSQL_DATABASE=myapp")
	}
	if strings.Contains(out, "MYSQL_DB=") {
		t.Error("MySQL should use MYSQL_DATABASE not MYSQL_DB")
	}
}

func TestGenerateEnv_MariaDB(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "mariadb", registry.DBConfig{
		Host: "localhost", Port: 3306, User: "root", Password: "root", Name: "myapp",
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "MARIADB_HOST=localhost") {
		t.Error("expected MARIADB_HOST=localhost")
	}
	if !strings.Contains(out, "MARIADB_DATABASE=myapp") {
		t.Error("expected MARIADB_DATABASE=myapp")
	}
}

func TestGenerateEnv_MongoDB(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "mongodb", registry.DBConfig{
		Host: "localhost", Port: 27017, User: "root", Password: "root",
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "MONGODB_HOST=localhost") {
		t.Error("expected MONGODB_HOST=localhost")
	}
	if !strings.Contains(out, "MONGODB_USER=root") {
		t.Error("expected MONGODB_USER=root")
	}
	if strings.Contains(out, "MONGODB_DB=") {
		t.Error("MongoDB should not emit a DB name var")
	}
}

func TestGenerateEnv_Redis(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "redis", registry.DBConfig{
		Host: "localhost", Port: 6379,
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "REDIS_HOST=localhost") {
		t.Error("expected REDIS_HOST=localhost")
	}
	if !strings.Contains(out, "REDIS_PORT=6379") {
		t.Error("expected REDIS_PORT=6379")
	}
	if strings.Contains(out, "REDIS_USER=") {
		t.Error("Redis should not emit auth vars")
	}
}

func TestGenerateEnv_SQLite(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "sqlite", registry.DBConfig{
		Path: "./data/app.db", SQLite: true,
	})
	out := wiring.GenerateEnv(ctx)
	if strings.Contains(out, "SQLITE_HOST") {
		t.Error("SQLite env should not contain host var")
	}
	if !strings.Contains(out, "DB_PATH=./data/app.db") {
		t.Error("expected DB_PATH")
	}
}

func TestGenerateEnv_PostgresAndRedis(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	ctx = addDB(ctx, "redis", registry.DBConfig{
		Host: "localhost", Port: 6379,
	})
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "POSTGRES_HOST=localhost") {
		t.Error("expected POSTGRES_HOST")
	}
	if !strings.Contains(out, "REDIS_HOST=localhost") {
		t.Error("expected REDIS_HOST")
	}
	if strings.Contains(out, "DB_HOST=") {
		t.Error("multi-DB should not emit legacy DB_HOST")
	}
}

func TestGenerateEnv_DockerMode(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "db", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	ctx.EnvMode = "docker"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "POSTGRES_HOST=db") {
		t.Error("docker mode should set POSTGRES_HOST=db")
	}
}

func TestGenerateEnv_NoDatabase(t *testing.T) {
	ctx := makeEnvCtx()
	out := wiring.GenerateEnv(ctx)
	if strings.Contains(out, "DB_") || strings.Contains(out, "POSTGRES_") || strings.Contains(out, "REDIS_") {
		t.Error("no DB selected — env should not contain any DB vars")
	}
}

func TestGenerateEnv_FrontendOnly(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendID:   "react",
		FrontendPort: 5173,
		EnvMode:      "local",
	}
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "FRONTEND_PORT=5173") {
		t.Error("expected FRONTEND_PORT")
	}
	if strings.Contains(out, "BACKEND_PORT") {
		t.Error("frontend-only should not contain BACKEND_PORT")
	}
	if strings.Contains(out, "VITE_API_URL") {
		t.Error("frontend-only should not contain VITE_API_URL")
	}
}

func TestGenerateEnv_DatabaseURL_Postgres(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	ctx.ORMID = "prisma"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "DATABASE_URL=postgresql://postgres:postgres@localhost:5432/myapp") {
		t.Errorf("expected postgres DATABASE_URL, got:\n%s", out)
	}
}

func TestGenerateEnv_DatabaseURL_MySQL(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "mysql", registry.DBConfig{
		Host: "localhost", Port: 3306, User: "root", Password: "root", Name: "myapp",
	})
	ctx.ORMID = "drizzle"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "DATABASE_URL=mysql://root:root@localhost:3306/myapp") {
		t.Errorf("expected mysql DATABASE_URL, got:\n%s", out)
	}
}

func TestGenerateEnv_DatabaseURL_MariaDB(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "mariadb", registry.DBConfig{
		Host: "localhost", Port: 3306, User: "root", Password: "root", Name: "myapp",
	})
	ctx.ORMID = "drizzle"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "DATABASE_URL=mysql://root:root@localhost:3306/myapp") {
		t.Errorf("expected mariadb DATABASE_URL using mysql:// scheme, got:\n%s", out)
	}
}

func TestGenerateEnv_DatabaseURL_SQLite(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "sqlite", registry.DBConfig{
		Path: "./data/app.db", SQLite: true,
	})
	ctx.ORMID = "prisma"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "DATABASE_URL=file:./data/app.db") {
		t.Errorf("expected sqlite DATABASE_URL with file: prefix, got:\n%s", out)
	}
}

func TestGenerateEnv_NoORM_NoDatabaseURL(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	out := wiring.GenerateEnv(ctx)
	if strings.Contains(out, "DATABASE_URL=") {
		t.Error("expected no DATABASE_URL when no ORM selected")
	}
}

func TestGenerateEnv_DatabaseURL_FirstSQLOnly(t *testing.T) {
	ctx := addDB(makeEnvCtx(), "postgres", registry.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Name: "myapp",
	})
	ctx = addDB(ctx, "redis", registry.DBConfig{Host: "localhost", Port: 6379})
	ctx.ORMID = "prisma"
	out := wiring.GenerateEnv(ctx)
	if !strings.Contains(out, "DATABASE_URL=postgresql://postgres:postgres@localhost:5432/myapp") {
		t.Errorf("expected postgres DATABASE_URL (first SQL), got:\n%s", out)
	}
	count := strings.Count(out, "DATABASE_URL=")
	if count != 1 {
		t.Errorf("expected exactly 1 DATABASE_URL line, got %d", count)
	}
}
