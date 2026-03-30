package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestGeneratePrismaSchema_Postgres(t *testing.T) {
	out, err := wiring.GeneratePrismaSchema("postgresql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `provider = "postgresql"`) {
		t.Error("expected postgresql provider")
	}
	if strings.Contains(out, "DATABASE_URL") {
		t.Error("schema.prisma must not contain DATABASE_URL in Prisma v7 — use prisma.config.ts")
	}
	if !strings.Contains(out, `provider = "prisma-client"`) {
		t.Error("expected generator prisma-client")
	}
	if !strings.Contains(out, `output   = "../src/generated/prisma"`) {
		t.Error("expected output path")
	}
}

func TestGeneratePrismaConfig(t *testing.T) {
	out := wiring.GeneratePrismaConfig()
	if !strings.Contains(out, "prisma/config") {
		t.Error("expected prisma/config import")
	}
	if !strings.Contains(out, "DATABASE_URL") {
		t.Error("expected DATABASE_URL reference in prisma.config.ts")
	}
	if !strings.Contains(out, "prisma/schema.prisma") {
		t.Error("expected schema path")
	}
	if !strings.Contains(out, "prisma/migrations") {
		t.Error("expected migrations path")
	}
}

func TestGeneratePrismaSchema_MySQL(t *testing.T) {
	out, err := wiring.GeneratePrismaSchema("mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `provider = "mysql"`) {
		t.Error("expected mysql provider")
	}
}

func TestGeneratePrismaSchema_SQLite(t *testing.T) {
	out, err := wiring.GeneratePrismaSchema("sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `provider = "sqlite"`) {
		t.Error("expected sqlite provider")
	}
}

func TestGenerateDrizzleConfig_Postgres(t *testing.T) {
	cfg, idx, err := wiring.GenerateDrizzleConfig("postgresql", "node-postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg, "dialect: 'postgresql'") {
		t.Error("expected dialect postgresql")
	}
	if !strings.Contains(cfg, "DATABASE_URL") {
		t.Error("expected DATABASE_URL in drizzle config")
	}
	if !strings.Contains(idx, "drizzle-orm/node-postgres") {
		t.Error("expected drizzle-orm/node-postgres import")
	}
	if strings.Contains(idx, "better-sqlite3") {
		t.Error("postgres index.ts should not contain better-sqlite3")
	}
}

func TestGenerateDrizzleConfig_MySQL(t *testing.T) {
	cfg, idx, err := wiring.GenerateDrizzleConfig("mysql", "mysql2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg, "dialect: 'mysql'") {
		t.Error("expected dialect mysql")
	}
	if !strings.Contains(idx, "drizzle-orm/mysql2") {
		t.Error("expected drizzle-orm/mysql2 import")
	}
}

func TestGenerateDrizzleConfig_SQLite(t *testing.T) {
	cfg, idx, err := wiring.GenerateDrizzleConfig("sqlite", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cfg, "dialect: 'sqlite'") {
		t.Error("expected dialect sqlite")
	}
	if !strings.Contains(idx, "drizzle-orm/better-sqlite3") {
		t.Error("expected better-sqlite3 import")
	}
	if !strings.Contains(idx, "{ uri: true }") {
		t.Error("sqlite index.ts must pass { uri: true }")
	}
	if strings.Contains(idx, "drizzle(process.env.DATABASE_URL!)") {
		t.Error("sqlite should not use URL string form of drizzle()")
	}
}
