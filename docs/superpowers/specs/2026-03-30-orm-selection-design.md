# ORM Selection Design

**Date:** 2026-03-30
**Status:** Approved

## Overview

Add an optional ORM selection step (Prisma or Drizzle) after database selection. The step is shown only when the project is ORM-eligible. Eligible projects receive generated config files and post-scaffold install instructions. No ORM integration is added to backend runtimes other than Node.js.

---

## 1. Eligibility Rules

The ORM step is shown when **all three** conditions are true:

1. **At least one SQL database selected** — `postgres`, `mysql`, `mariadb`, or `sqlite`. Redis-only and MongoDB-only projects are not eligible.
2. **Node/TS runtime present** — either a Node-based backend (`runtime == "node"`), including `backend-only` mode, or a server-side frontend-only project (`OutputMode == "frontend-only"` with a server-side capable framework).
3. **Server-side capable framework** — for full-stack or backend-only, any Node backend qualifies. For frontend-only, only frameworks with `ServerSide: true` qualify (Next.js, Astro, SvelteKit, TanStack Start). Plain SPAs (React, Vue, Angular) do not qualify.

A helper function `isORMEligible(m Model) bool` in `model.go` encodes all three rules.

If the project is not eligible, `phaseORMSelect` is skipped entirely and the flow advances directly to `phaseEnvMode`.

---

## 2. Registry Entry Changes

Add `ServerSide bool` field to the `Entry` struct in `internal/registry/types.go`:

```go
ServerSide bool `yaml:"server_side"` // true for frontend frameworks with a server runtime (Next.js, Astro, SvelteKit, TanStack Start)
```

Set `server_side: true` in the YAML for:
- `nextjs.yaml`, `nextjs-bun.yaml`
- `astro.yaml`, `astro-bun.yaml`
- `svelte.yaml`, `svelte-bun.yaml` (SvelteKit)
- `tanstack.yaml`, `tanstack-bun.yaml`

All other frontend YAML files leave `server_side` unset (defaults to `false`).

---

## 3. WeldContext Changes

Add one field to `WeldContext` in `internal/registry/types.go`:

```go
ORMID string // "prisma", "drizzle", or "" for none — no yaml tag needed; WeldContext is not deserialized from YAML
```

---

## 4. TUI Phase

### New phase constant

```go
phaseORMSelect phase = iota // inserted between phaseDatabaseSelect and phaseEnvMode
```

Phase order (updated):
```
phaseProjectName
phaseOutputStructure
phaseFrontendRuntime
phaseFrontendFramework
phaseBackendRuntime
phaseBackendFramework
phaseDatabaseSelect
phaseORMSelect          ← new
phaseEnvMode
phasePortOverrides
phaseConfirm
phaseExecuting
phaseDone
```

### Phase handler

The `phaseDatabaseSelect` handler advances to `phaseORMSelect`. The `phaseORMSelect` handler:

```go
case phaseORMSelect:
    m.ctx.ORMID = ormIDFromName(msg.Value.(string))
    m.phase = phaseEnvMode
    m.current = steps.NewEnvMode()
```

Where `ormIDFromName` maps `"Prisma"` → `"prisma"`, `"Drizzle"` → `"drizzle"`, `"None"` → `""`.

### Guard in `phaseDatabaseSelect` handler

After setting `DatabaseIDs`/`DBConfigs`, instead of always advancing to `phaseORMSelect`:

```go
if isORMEligible(m) {
    m.phase = phaseORMSelect
    m.current = steps.NewRuntimeSelect("Select ORM (optional):", ormOptions())
} else {
    m.phase = phaseEnvMode
    m.current = steps.NewEnvMode()
}
```

### `ormOptions()` helper

```go
func ormOptions() []steps.RuntimeOption {
    return []steps.RuntimeOption{
        {Name: "None", Available: true},
        {Name: "Prisma", Available: true},
        {Name: "Drizzle", Available: true},
    }
}
```

Reuses the existing `RuntimeSelect` component (single-select, Enter to confirm). `"None"` is first and is therefore the default cursor position — pressing Enter without navigating selects None (`ORMID = ""`). This is intentional: ORM is opt-in.

### `isORMEligible` helper

```go
func isORMEligible(m Model) bool {
    // Rule 1: at least one SQL database selected
    sqlDBs := map[string]bool{"postgres": true, "mysql": true, "mariadb": true, "sqlite": true}
    hasSQL := false
    for _, id := range m.ctx.DatabaseIDs {
        if sqlDBs[id] {
            hasSQL = true
            break
        }
    }
    if !hasSQL {
        return false
    }

    // Rule 2 + 3: Node backend always qualifies; frontend-only requires ServerSide
    if m.ctx.BackendID != "" {
        for _, e := range m.entries {
            if e.ID == m.ctx.BackendID && e.Runtime == "node" {
                return true
            }
        }
        return false
    }
    // frontend-only
    if m.ctx.FrontendID != "" {
        for _, e := range m.entries {
            if e.ID == m.ctx.FrontendID && e.ServerSide {
                return true
            }
        }
    }
    return false
}
```

---

## 5. Generated Files

Files are written in `main.go` after the `.env` write, only when `ctx.ORMID != ""`.

### Target directory (`serviceDir`)

The ORM files are written into the Node service directory. `serviceDir` resolves as follows:

| Output mode | `serviceDir` |
|---|---|
| `monorepo` | `{projectRoot}/backend` |
| `separate` | `{projectName}-backend` (sibling of cwd, no shared root) |
| `frontend-only` | `{projectRoot}/frontend` (always has a project root; no separate sub-mode exists) |
| `backend-only` | `{projectRoot}/backend` |

This mirrors how `main.go` already resolves `frontendDir` and `backendDir` per output mode.

### `DATABASE_URL` in `.env`

Prisma and Drizzle both expect a single `DATABASE_URL` connection string. The multi-DB `.env` format uses prefixed individual vars (`POSTGRES_HOST`, `POSTGRES_PORT`, etc.) and does not emit `DATABASE_URL`.

When `ctx.ORMID != ""`, `GenerateEnv` emits one additional line — a composed `DATABASE_URL` for the primary SQL database — appended after the prefixed vars:

| DB | `DATABASE_URL` format |
|---|---|
| `postgres` | `postgresql://{{POSTGRES_USER}}:{{POSTGRES_PASSWORD}}@{{POSTGRES_HOST}}:{{POSTGRES_PORT}}/{{POSTGRES_DB}}` |
| `mysql` / `mariadb` | `mysql://{{MYSQL_USER}}:{{MYSQL_PASSWORD}}@{{MYSQL_HOST}}:{{MYSQL_PORT}}/{{MYSQL_DATABASE}}` (or `MARIADB_*` prefix) |
| `sqlite` | `file:{{DB_PATH}}` — Prisma requires the `file:` URI scheme; `better-sqlite3` supports it via the `{ uri: true }` option (see Drizzle `index.ts` below) |

The Drizzle sqlite `src/db/index.ts` template passes `{ uri: true }` to `better-sqlite3` to handle the `file:` prefix:

```typescript
import 'dotenv/config';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import Database from 'better-sqlite3';

const db = drizzle(new Database(process.env.DATABASE_URL!, { uri: true }));

export { db };
```

The values come from `ctx.DBConfigs[primaryDBID]` — the same data already used to generate the prefixed vars. The `GenerateEnv` signature does not change; it already receives the full `WeldContext` including `ORMID`.

**Only one `DATABASE_URL` is emitted** — for the first SQL DB in `DatabaseIDs`. If multiple SQL DBs are selected (e.g. postgres + mysql), only the first gets `DATABASE_URL` since ORMs connect to a single database.

### SQL database provider mapping

For both Prisma and Drizzle, the "primary SQL database" is the first entry in `ctx.DatabaseIDs` that is a SQL DB (postgres, mysql, mariadb, sqlite). Prisma supports only one datasource; Drizzle config is per-dialect.

| `DatabaseID` | Prisma provider | Drizzle dialect | Drizzle driver pkg |
|---|---|---|---|
| `postgres` | `postgresql` | `postgresql` | `pg`, `@types/pg` |
| `mysql` | `mysql` | `mysql` | `mysql2` |
| `mariadb` | `mysql` | `mysql` | `mysql2` |
| `sqlite` | `sqlite` | `sqlite` | `better-sqlite3`, `@types/better-sqlite3` |

### Prisma

**Target: Prisma 7+** (uses `provider = "prisma-client"`; Prisma 6 users should upgrade).

File: `{serviceDir}/prisma/schema.prisma`

```prisma
generator client {
  provider = "prisma-client"
  output   = "../src/generated/prisma"
}

datasource db {
  provider = "{{.PrismaProvider}}"
  url      = env("DATABASE_URL")
}
```

The `output` path `"../src/generated/prisma"` is relative to the `prisma/` subdirectory, resolving to `{serviceDir}/src/generated/prisma` — consistent with the official Prisma 7 examples.

Template data struct:
```go
type prismaTemplateData struct {
    PrismaProvider string // "postgresql", "mysql", or "sqlite"
}
```

### Drizzle

Two files:

**`{serviceDir}/drizzle.config.ts`**:
```typescript
import 'dotenv/config';
import { defineConfig } from 'drizzle-kit';

export default defineConfig({
  out: './drizzle',
  schema: './src/db/schema.ts',
  dialect: '{{.DrizzleDialect}}',
  dbCredentials: {
    url: process.env.DATABASE_URL!,
  },
});
```

**`{serviceDir}/src/db/index.ts`** — two variants depending on the DB:

For postgres and mysql/mariadb (URL string form):
```typescript
import 'dotenv/config';
import { drizzle } from 'drizzle-orm/{{.DrizzleImport}}';

const db = drizzle(process.env.DATABASE_URL!);

export { db };
```

For sqlite (`better-sqlite3` requires a `Database` instance; `{ uri: true }` is needed because `DATABASE_URL` uses the `file:` prefix):
```typescript
import 'dotenv/config';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import Database from 'better-sqlite3';

const db = drizzle(new Database(process.env.DATABASE_URL!, { uri: true }));

export { db };
```

The `GenerateDrizzleConfig` function returns the appropriate `index.ts` content based on the dialect — it is not a single shared template.

Template data struct:
```go
type drizzleTemplateData struct {
    DrizzleDialect string // "postgresql", "mysql", "sqlite"
    DrizzleImport  string // "node-postgres", "mysql2" (sqlite uses hardcoded template)
}
```

| DB | `DrizzleDialect` | `DrizzleImport` | `index.ts` variant |
|---|---|---|---|
| postgres | `postgresql` | `node-postgres` | URL string |
| mysql / mariadb | `mysql` | `mysql2` | URL string |
| sqlite | `sqlite` | `better-sqlite3` | `new Database(...)` |

Templates are embedded Go template strings (inline in `wiring/orm.go`, not separate files — they are short enough).

---

## 6. Wiring Package: `wiring/orm.go`

New file `internal/wiring/orm.go` with two exported functions:

```go
// GeneratePrismaSchema returns the content of prisma/schema.prisma.
func GeneratePrismaSchema(provider string) (string, error)

// GenerateDrizzleConfig returns (drizzle.config.ts content, src/db/index.ts content, error).
// For dialect == "sqlite", importPath is ignored — the sqlite index.ts template is hardcoded.
// Callers should pass importPath = "" for sqlite.
func GenerateDrizzleConfig(dialect, importPath string) (string, string, error)
```

Both use `text/template` with the template strings defined as package-level constants.

---

## 7. Post-Scaffold Instructions

Appended to the next-steps output in `main.go` after the standard "Next steps:" block.

**Prisma:**
```
ORM: Prisma
  npm install prisma @prisma/client
  npx prisma generate
```

**Drizzle (postgres example):**
```
ORM: Drizzle
  npm install drizzle-orm pg dotenv
  npm install -D drizzle-kit tsx @types/pg
```

Install commands are derived from the primary SQL DB using the driver mapping table in Section 5.

A helper `ormInstallInstructions(ormID, primaryDBID string) string` in `main.go` returns the formatted block.

`dotenv` is always included in Drizzle runtime dependencies regardless of DB, because `drizzle.config.ts` always uses `import 'dotenv/config'`. The install commands per DB are:

| DB | Drizzle runtime deps | Drizzle dev deps |
|---|---|---|
| postgres | `drizzle-orm pg dotenv` | `drizzle-kit tsx @types/pg` |
| mysql / mariadb | `drizzle-orm mysql2 dotenv` | `drizzle-kit tsx` |
| sqlite | `drizzle-orm better-sqlite3 dotenv` | `drizzle-kit tsx @types/better-sqlite3` |

---

## 8. Summary Screen

Add one line to `summaryLines()`:

```
ORM:        Prisma
```
or `none` when `ORMID == ""`.

---

## 9. Testing

New file: `internal/wiring/orm_test.go`

Test cases:
- `TestGeneratePrismaSchema_Postgres` — verify `provider = "postgresql"` and `url = env("DATABASE_URL")`
- `TestGeneratePrismaSchema_MySQL` — verify `provider = "mysql"`
- `TestGeneratePrismaSchema_SQLite` — verify `provider = "sqlite"`
- `TestGenerateDrizzleConfig_Postgres` — verify `dialect: 'postgresql'` and `drizzle-orm/node-postgres`
- `TestGenerateDrizzleConfig_MySQL` — verify `dialect: 'mysql'` and `drizzle-orm/mysql2`
- `TestGenerateDrizzleConfig_SQLite` — verify `dialect: 'sqlite'` and `drizzle-orm/better-sqlite3`
- `TestIsORMEligible` — unit tests for the eligibility helper (SQL DB + node backend, SQL DB + server-side frontend, SQL DB + SPA → false, no SQL DB + node backend → false, redis-only → false)

---

## 10. WordPress Mode

WordPress projects bypass `phaseDatabaseSelect` entirely (the WordPress handler in `phaseOutputStructure` sets `DatabaseIDs`/`DBConfigs` directly and jumps to `phasePortOverrides`). Therefore `phaseORMSelect` is never reached for WordPress projects. No guard is needed in `isORMEligible` — the phase is structurally unreachable.

---

## 11. Out of Scope

- MongoDB ORM support (Prisma supports it but requires a completely different schema approach)
- Per-ORM schema model generation (just the config file, not example models)
- ORM migration commands in next-steps beyond `prisma generate`
- Non-Node ORMs (SQLAlchemy, GORM, etc.)
