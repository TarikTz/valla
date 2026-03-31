# Fully Dockerized Dev Environment — Design Spec

**Date:** 2026-03-31
**Status:** Approved

---

## Overview

Add a "Fully Dockerized" output structure option to the valla TUI. When selected, valla scaffolds the chosen stack without requiring any local runtime (Node, Go, Python, etc.) — only Docker is needed. It generates a VS Code Dev Container config, a dev-focused Docker Compose file, and a Makefile, giving both IDE users and terminal-only users a zero-friction entry point.

---

## Struct Changes

### `WeldContext` (internal/registry/types.go)

Add one new field:

```go
DevContainer bool // true when "Fully Dockerized" output structure is chosen
```

`DevContainer == true` and `OutputMode == "devcontainer"` are always assigned together and are equivalent discriminators. All downstream conditionals use `ctx.DevContainer`.

### `Model` (internal/tui/model.go)

Add two new fields alongside the existing `feRuntimeOpts`/`beRuntimeOpts`:

```go
allFeRuntimeOpts []steps.RuntimeOption // all Available: true — used on DevContainer path
allBeRuntimeOpts []steps.RuntimeOption // all Available: true — used on DevContainer path
```

These are populated in `tui.New` and remain constant for the lifetime of the model.

---

## TUI Flow

### New output structure option

`"Fully Dockerized"` is added to the `OutputStructure` step alongside Monorepo, Separate folders, Frontend only, Backend only, and WordPress.

- The option is only shown if `docker` is detected. See Detector Changes below for the exact detection approach.
- `tui.New` receives a new `dockerAvailable bool` parameter. Updated signature:
  ```go
  func New(entries []registry.Entry, feRuntimeOpts, beRuntimeOpts []steps.RuntimeOption, dockerAvailable bool) Model
  ```
  `dockerAvailable` is stored on the `Model` struct and passed to `steps.NewOutputStructure(dockerAvailable bool)`, which stores it on the `OutputStructure` struct and uses it in `View()` to conditionally append `"Fully Dockerized"` to the options slice.
- In `tui.New`, compute and store the all-available slices:
  ```go
  return Model{
      ...
      feRuntimeOpts:    feRuntimeOpts,
      beRuntimeOpts:    beRuntimeOpts,
      allFeRuntimeOpts: steps.AllRuntimeOptions(feRuntimeOpts),
      allBeRuntimeOpts: steps.AllRuntimeOptions(beRuntimeOpts),
  }
  ```
- The directory tree preview updates to show a monorepo layout with `.devcontainer/` when "Fully Dockerized" is highlighted.

### Output mode and layout

Fully Dockerized always uses a monorepo layout. `WeldContext.OutputMode = "devcontainer"` and `WeldContext.DevContainer = true` are always assigned together.

### phaseOutputStructure handler changes

The existing runtime availability guard in the `phaseOutputStructure` handler:
```go
if !anyAvailable(m.feRuntimeOpts) || !anyAvailable(m.beRuntimeOpts) {
    return m, func() tea.Msg { return ErrMsg{...} }
}
```
is skipped on the DevContainer path. The handler checks `choice == "Fully Dockerized"` **before** this guard. When "Fully Dockerized" is chosen:

1. Set `ctx.OutputMode = "devcontainer"`, `ctx.DevContainer = true`, `ctx.EnvMode = "docker"`.
2. Set `cfg.Host = "db"` for all databases in `ctx.DBConfigs`.
3. **Swap the runtime option slices on the model:**
   ```go
   m.feRuntimeOpts = m.allFeRuntimeOpts
   m.beRuntimeOpts = m.allBeRuntimeOpts
   ```
   This makes all downstream phase handlers (`phaseFrontendRuntime`, `phaseBackendRuntime`) automatically use the all-available slices without any additional conditionals in those handlers.
4. Skip the runtime availability guard entirely.
5. Transition to `phaseFrontendRuntime` as normal (using `m.feRuntimeOpts`, which now holds the all-available slice).

### `AllRuntimeOptions` helper

New exported function in `internal/tui/steps/runtime_select.go` (alongside the existing unexported `runtimeOptionsAll`):

```go
// AllRuntimeOptions returns a copy of opts with every entry marked Available: true.
// The input slice must be the full set of known runtimes (not a pre-filtered subset).
func AllRuntimeOptions(opts []RuntimeOption) []RuntimeOption {
    out := make([]RuntimeOption, len(opts))
    for i, o := range opts {
        o.Available = true
        out[i] = o
    }
    return out
}
```

### ORM selection

`isORMEligible` is unchanged — ORM is only offered for Node backends, same as today. Applies equally in the DevContainer path.

### Phase transition table (changes only)

| Phase | Existing next phase | New next phase when `ctx.DevContainer == true` |
|-------|--------------------|--------------------------------------------|
| `phaseORMSelect` | `phaseEnvMode` | `phasePortOverrides` |
| `phaseDatabaseSelect` (no-ORM else branch) | `phaseEnvMode` | `phasePortOverrides` |

`phaseEnvMode` is never entered when `ctx.DevContainer == true`.

### Summary and success output changes

**`summaryLines()` in `model.go`:** The `Env mode: %s` line is emitted unconditionally for non-WordPress paths. When `ctx.DevContainer == true`, replace this line with `"Mode: Fully Dockerized (Dev Container)"`.

**`renderSummaryCard()` in `main.go`:** Add a new top-level branch for `ctx.DevContainer == true`, parallel to the existing `ctx.OutputMode == "wordpress"` branch. It renders the mode row as `"Dev Container (Docker)"` plus the standard frontend/backend/database/ORM rows (same as the existing `else` block). This follows the same pattern as the WordPress branch.

**`renderSuccessOutput()` in `main.go`:** Add `ctx.DevContainer` as the **outermost condition** before the existing `EnvMode` branches. Without this, the `else if ctx.EnvMode == "docker"` branch fires (since `ctx.EnvMode = "docker"`), incorrectly showing `docker-compose up -d`:

```go
if ctx.DevContainer {
    // tree block (see below)
    // commands: cd <project> && make dev
    // secondary: open in VS Code → "Reopen in Container"
} else if ctx.EnvMode == "local" {
    ...
} else if ctx.EnvMode == "docker" {
    ...
}
```

The directory-tree block for `ctx.DevContainer == true`:
```
<projectName>/
├── frontend/
├── backend/
├── .devcontainer/
│   └── devcontainer.json
├── .env
├── docker-compose.yml
├── docker-compose.dev.yml
└── Makefile
```

---

## Generated Files

When `ctx.DevContainer == true`, three additional files are generated after the standard scaffolding output. All paths are under `projectRoot/` (monorepo layout).

### Runtime → devcontainer image mapping

Used for both `devcontainer.json` and `docker-compose.dev.yml`. Keys are exact `Entry.Runtime` values from the registry:

| `Entry.Runtime` | Image |
|-----------------|-------|
| `go` | `mcr.microsoft.com/devcontainers/go` |
| `node` | `mcr.microsoft.com/devcontainers/javascript-node` |
| `bun` | `mcr.microsoft.com/devcontainers/javascript-node` |
| `python3` | `mcr.microsoft.com/devcontainers/python` |
| `java` | `mcr.microsoft.com/devcontainers/java` |
| `dotnet` | `mcr.microsoft.com/devcontainers/dotnet` |

Note: Python registry entries use `runtime: python3` (not `python`). Bun maps to the Node devcontainer image.

### `.devcontainer/devcontainer.json`

- Image derived from `backendEntry.Runtime` via the mapping table above. `backendEntry.DevContainerImage` overrides if set.
- References `../docker-compose.dev.yml` as its compose file.
- Port forwarding via `WeldContext` fields:
  ```json
  "forwardPorts": [{{.FrontendPort}}, {{.BackendPort}}]
  ```
- VS Code extensions (keys are `Entry.Runtime` values):

| Runtime | Language extension ID | Always included |
|---------|-----------------------|-----------------|
| `go` | `golang.go` | `ms-azuretools.vscode-docker` |
| `node`, `bun` | `dbaeumer.vscode-eslint` | `ms-azuretools.vscode-docker` |
| `python3` | `ms-python.python` | `ms-azuretools.vscode-docker` |
| `java` | `vscjava.vscode-java-pack` | `ms-azuretools.vscode-docker` |
| `dotnet` | `ms-dotnettools.csharp` | `ms-azuretools.vscode-docker` |

### `docker-compose.dev.yml`

- No `build:` key. Each service uses its devcontainer image directly (from the mapping table):
  - `backend`: `backendEntry.Runtime` lookup, or `backendEntry.DevContainerImage` if set
  - `frontend`: `frontendEntry.Runtime` lookup
- Volume mounts: `./frontend:/app`, `./backend:/app`
- Each service runs `Entry.DevCmd` as its command
- Databases are identical to the production `docker-compose.yml`
- All Microsoft devcontainer images include `sh` at `/bin/sh`

### `Makefile`

| Target | Command |
|--------|---------|
| `dev` | `docker compose -f docker-compose.dev.yml up` |
| `down` | `docker compose -f docker-compose.dev.yml down` |
| `shell-frontend` | `docker compose -f docker-compose.dev.yml exec frontend sh` |
| `shell-backend` | `docker compose -f docker-compose.dev.yml exec backend sh` |
| `logs` | `docker compose -f docker-compose.dev.yml logs -f` |

All three files are Go templates in `internal/registry/data/templates/devcontainer/`, rendered via `ReadEmbeddedFile`. No changes to `loader.go` are needed — `yaml.Unmarshal` handles new optional `Entry` fields gracefully when absent from YAML.

---

## Registry Changes

Two optional new fields on `Entry` in `internal/registry/types.go`:

```go
DevContainerImage string `yaml:"devcontainer_image"` // overrides runtime-derived devcontainer image
DevCmd            string `yaml:"dev_cmd"`             // hot-reload command inside the dev container
```

`DevCmd` defaults by entry ID if not set in YAML:

| Entry ID pattern | Default DevCmd |
|-----------------|----------------|
| `go-*` | `go run .` |
| `node-*` | `npm run dev` |
| `python-fastapi` | `uvicorn main:app --reload --host 0.0.0.0` |
| `python-flask` | `flask run --host 0.0.0.0` |
| `python-django` | `python manage.py runserver 0.0.0.0:8000` |
| `java-springboot-maven` | `./mvnw spring-boot:run` |
| `java-springboot-gradle` | `./gradlew bootRun` |
| `java-quarkus-maven` | `./mvnw quarkus:dev` |
| `java-quarkus-gradle` | `./gradlew quarkusDev` |
| `dotnet-*` | `dotnet watch run` |

All four Java YAML registry files must set `dev_cmd` explicitly. The Go-level default is a fallback only.

---

## Scaffolder Changes

New file `internal/scaffolder/scaffolder_devcontainer.go`:

```go
func GenerateDevContainerFiles(ctx registry.WeldContext, backendEntry registry.Entry, frontendEntry registry.Entry, projectRoot string) error
```

- `ctx` — full weld context (provides `FrontendPort`, `BackendPort`, `DevContainer`, etc.)
- `backendEntry` — resolved backend `Entry` (for `DevContainerImage`, `DevCmd`, `Runtime`)
- `frontendEntry` — resolved frontend `Entry` (for `Runtime`, to derive frontend image)
- `projectRoot` — absolute path to project root

Called from `cmd/valla/main.go` after existing scaffolding completes, when `ctx.DevContainer == true`.

**Rollback on error:** use `os.RemoveAll` on `.devcontainer/` (directory); use `os.Remove` on `docker-compose.dev.yml` and `Makefile` (flat files).

**`.env` generation:** `ctx.EnvMode = "docker"` so `wiring.GenerateEnv(ctx)` runs normally and produces a Docker-flavored `.env` (`DB_HOST=db`, etc.). Correct — databases are addressed by service name in the dev compose file.

`scaffolder.go` is not modified.

---

## Detector Changes

Add `"docker"` to the **existing** `detector.Detect` call in `cmd/valla/main.go`. The java detection via `DetectWithAliases` is **unchanged**:

```go
// Before:
available := detector.Detect([]string{"go", "node", "bun", "python3", "dotnet"})

// After:
available := detector.Detect([]string{"go", "node", "bun", "python3", "dotnet", "docker"})
dockerAvailable := available["docker"]
```

The `DetectWithAliases` block for java remains exactly as-is. `dockerAvailable` is passed to `tui.New(entries, feRuntimeOpts, beRuntimeOpts, dockerAvailable)`.

---

## Out of Scope

- Multi-service architecture (multiple frontends or backends) — deferred
- Non-Docker local dev environment generation
- CI/CD configuration generation
