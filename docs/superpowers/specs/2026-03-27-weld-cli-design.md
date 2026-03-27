# weld-cli Design Spec

**Date:** 2026-03-27
**Status:** Approved

---

## Overview

`weld-cli` is an interactive, lightning-fast CLI tool built in Go that scaffolds full-stack web applications. It detects available runtimes on the user's machine, guides them through a bubbletea TUI to select their stack, runs official scaffolding commands for each piece, then wires the pieces together with CORS config, environment files, and Docker Compose.

---

## Goals

- Reduce project initialization from hours to seconds
- Use official scaffolders (not custom templates) so generated projects are always up-to-date
- Stay extensible: adding a new framework requires only a new registry YAML entry, no core changes
- Ship v1 as a binary distributed via GitHub Releases + NPX wrapper

---

## v1 Supported Matrix

| Type     | Options                     |
|----------|-----------------------------|
| Frontend | React, Vue                  |
| Backend  | Go (Gin, Fiber, Boilerplate), Node.js (Express, Nest, Boilerplate) |
| Database | PostgreSQL, SQLite           |

**8 base combinations.** Extensible to full matrix in future releases.

---

## Architecture

weld-cli is composed of four layers:

```
┌─────────────────────────────────────────┐
│           TUI Layer (bubbletea)         │  ← prompts, spinners, output
├─────────────────────────────────────────┤
│         Orchestrator (core CLI)         │  ← runtime detection, flow control
├─────────────────────────────────────────┤
│         Registry (YAML config)          │  ← stack definitions, defaults
├─────────────────────────────────────────┤
│      Scaffolder + Wiring Engine         │  ← runs commands, injects config
└─────────────────────────────────────────┘
```

### Registry

YAML files embedded into the binary via `go:embed`. Each entry describes one stack variant:

```yaml
# registry/backends/go-gin.yaml
id: go-gin
name: Go (Gin)
type: backend
runtime: go
group: Go
requires_runtime: go
scaffold_cmd: "go mod init {{.ProjectName}} && go get github.com/gin-gonic/gin"
default_port: 8080
env_vars:
  - PORT
  - DB_HOST
  - DB_PORT
  - DB_USER
  - DB_PASSWORD
  - DB_NAME
cors_origins: ["http://localhost:{{.FrontendPort}}"]
cors_patch:
  file: main.go
  template: |
    r.Use(cors.New(cors.Config{
        AllowOrigins: []string{"http://localhost:{{.FrontendPort}}"},
    }))
docker:
  image: golang:1.22-alpine
  build_context: ./backend
```

```yaml
# registry/frontends/react.yaml
id: react
name: React
type: frontend
runtime: node
group: React
requires_runtime: node
scaffold_cmd: "npm create vite@latest {{.ProjectName}} -- --template react"
default_port: 5173
http_client_patch:
  file: src/api.ts
  template: |
    export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:{{.BackendPort}}'
```

```yaml
# registry/databases/postgres.yaml
id: postgres
name: PostgreSQL
type: database
default_port: 5432
docker:
  image: postgres:16-alpine
  env_vars:
    POSTGRES_USER: "{{.DBUser}}"
    POSTGRES_PASSWORD: "{{.DBPassword}}"
    POSTGRES_DB: "{{.DBName}}"
```

Key properties:
- `group` — groups framework variants under the same runtime for two-step selection
- `requires_runtime` — used by the detector to filter available options
- `default_port` — shown in port override step, user can change
- `scaffold_cmd` — Go template string executed during scaffolding
- `cors_patch` / `http_client_patch` — targeted file patches, not full-file replacements

---

## Prompt Flow

```
1. Project name          → text input
2. Output structure      → Monorepo / Separate folders
3. Frontend runtime      → filtered to installed runtimes
4. Frontend framework    → filtered to selected runtime group
5. Backend runtime       → filtered to installed runtimes
6. Backend framework     → filtered to selected runtime group
7. Database              → PostgreSQL, SQLite
8. Environment mode      → Local (.env) / Docker Compose
9. Port overrides        → tabbed form, all defaults shown, all editable
10. Confirm summary      → full review of selections before execution
```

Runtime detection runs at startup. Only runtimes found on `$PATH` are shown as options.

---

## Scaffolder

Executes `scaffold_cmd` for each selected stack entry using Go's `os/exec`. Commands run in sequence with a bubbletea spinner per step. On error, weld-cli stops, prints the failed command output, and cleans up any partially created directories.

---

## Wiring Engine

Three outputs, always generated after scaffolding:

### 1. Environment config

**Local mode** — generates `.env` at project root:

```
FRONTEND_PORT=5173
BACKEND_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=postgres
DB_PASSWORD=postgres
VITE_API_URL=http://localhost:8080
```

**Docker mode** — generates `docker-compose.yml` and `Dockerfile` per service, derived from registry `docker` entries.

### 2. CORS patch

Applies `cors_patch` from the backend registry entry — a small Go template injected at a specific file path in the generated backend. Configures allowed origins to point at the frontend port.

### 3. HTTP client patch

Applies `http_client_patch` from the frontend registry entry — injects a pre-configured API base URL constant pointing at the backend port.

Patches are file-path-specific and small — not full-file replacements — minimizing breakage when upstream scaffolders change their output structure.

---

## Output Structure

User selects one of:

- **Monorepo** — single root folder with `/frontend`, `/backend`, shared `.env` / `docker-compose.yml` at root
- **Separate folders** — root directory created, each service is self-contained inside it, linked only by env/docker config

---

## Completion Output

```
✓ Project scaffolded successfully!

Next steps:
  cd my-project

  # Local mode:
  npm install        (in /frontend)
  go run main.go     (in /backend)

  # Docker mode:
  docker-compose up -d
```

---

## Repository Structure

```
weld-cli/
├── cmd/
│   └── weld/
│       └── main.go
├── internal/
│   ├── registry/        # loads + queries YAML registry entries
│   ├── detector/        # runtime detection (go, node, etc.)
│   ├── tui/             # bubbletea models & views
│   ├── scaffolder/      # executes scaffold_cmds
│   └── wiring/          # env, docker-compose, CORS, HTTP client patches
├── registry/
│   ├── frontends/       # react.yaml, vue.yaml
│   ├── backends/        # go-gin.yaml, go-fiber.yaml, node-express.yaml, etc.
│   └── databases/       # postgres.yaml, sqlite.yaml
├── npm/
│   └── index.js         # NPX wrapper
├── .goreleaser.yaml
└── README.md
```

---

## Distribution (v1)

1. **GoReleaser** compiles binaries for macOS (ARM/Intel), Linux (x64/ARM), Windows (x64) and attaches them to GitHub Releases
2. **NPX wrapper** (`npm/index.js`) — detects OS/arch, downloads correct binary from GitHub Releases, caches it, execs it
3. User runs: `npx weld-cli`

Future: Homebrew tap, Chocolatey, `.deb`/`.rpm` packages.

---

## Extensibility

Adding a new framework to weld-cli:
1. Add a YAML file to `registry/backends/` (or `frontends/`, `databases/`)
2. No changes to core CLI logic required

The compatibility matrix (CORS + Docker wiring) is fully driven by registry entries — adding a new stack automatically participates in the wiring system.

---

## Non-Goals (v1)

- Full 4×5×4 matrix support (v2+)
- Homebrew / Chocolatey / Linux packages (v2+)
- Authentication scaffolding
- CI/CD pipeline generation
- Cloud deployment config
