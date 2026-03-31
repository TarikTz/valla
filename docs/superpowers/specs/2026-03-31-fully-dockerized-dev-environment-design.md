# Fully Dockerized Dev Environment — Design Spec

**Date:** 2026-03-31
**Status:** Approved

---

## Overview

Add a "Fully Dockerized" output structure option to the valla TUI. When selected, valla scaffolds the chosen stack without requiring any local runtime (Node, Go, Python, etc.) — only Docker is needed. It generates a VS Code Dev Container config, a dev-focused Docker Compose file, and a Makefile, giving both IDE users and terminal-only users a zero-friction entry point.

---

## TUI Flow

### New output structure option

`"Fully Dockerized"` is added to the `OutputStructure` step alongside Monorepo, Separate folders, Frontend only, Backend only, and WordPress.

- The option is **only shown if `docker` is detected on the machine**. Detection uses the existing `detector.Detect` mechanism; the result is passed into `tui.New` from the entrypoint.
- The directory tree preview on the right updates to show the devcontainer layout when this option is highlighted.

### Flow differences when selected

The user proceeds through the same single frontend + single backend + database selection as Monorepo, with two differences:

1. **Runtime detection is bypassed** — all frameworks are shown as available. Nothing needs to be installed locally since everything runs in Docker.
2. **`phaseEnvMode` is skipped** — the "How would you like to run this locally?" step is omitted. Mode is implicitly `docker`.

`phasePortOverrides` and `phaseConfirm` are unchanged.

### WeldContext change

One new field:

```go
DevContainer bool // true when "Fully Dockerized" output structure is chosen
```

---

## Generated Files

When `DevContainer == true`, three additional files are generated after the standard scaffolding output (`.env`, `docker-compose.yml`, frontend/backend source):

### `.devcontainer/devcontainer.json`

Uses the Microsoft devcontainer base image matched to the selected backend runtime:

| Runtime | Image |
|---------|-------|
| `go` | `mcr.microsoft.com/devcontainers/go` |
| `node` | `mcr.microsoft.com/devcontainers/javascript-node` |
| `python` | `mcr.microsoft.com/devcontainers/python` |
| `java` | `mcr.microsoft.com/devcontainers/java` |
| `dotnet` | `mcr.microsoft.com/devcontainers/dotnet` |

References `docker-compose.dev.yml` as its compose file, forwards frontend and backend ports, and includes recommended VS Code extensions (language server for the backend runtime + Docker extension).

### `docker-compose.dev.yml`

Dev-focused compose file:
- Source code mounted as volumes — no build step needed for iteration
- Each service runs its hot-reload dev command (`go run .`, `npm run dev`, `uvicorn --reload`, etc.)
- Databases are identical to the production compose file

### `Makefile`

Targets:

| Target | Command |
|--------|---------|
| `dev` | `docker compose -f docker-compose.dev.yml up` |
| `down` | `docker compose -f docker-compose.dev.yml down` |
| `shell-frontend` | `docker compose -f docker-compose.dev.yml exec frontend sh` |
| `shell-backend` | `docker compose -f docker-compose.dev.yml exec backend sh` |
| `logs` | `docker compose -f docker-compose.dev.yml logs -f` |

All three files are Go templates in `internal/registry/data/templates/devcontainer/` and rendered via the existing scaffolder using `WeldContext`.

---

## Registry Changes

Two optional new fields on `Entry`:

```go
DevContainerImage string `yaml:"devcontainer_image"` // overrides runtime-derived devcontainer image
DevCmd            string `yaml:"dev_cmd"`             // hot-reload command inside the dev container
```

`DevContainerImage` allows a specific stack to override the runtime default if needed (e.g. a stack that needs a specific Go version). `DevCmd` specifies the dev server command; falls back to a per-runtime default if not set:

| Runtime | Default DevCmd |
|---------|---------------|
| `go` | `go run .` |
| `node` | `npm run dev` |
| `python` | `uvicorn main:app --reload` |
| `java` | `./mvnw spring-boot:run` (Maven) / `./gradlew bootRun` (Gradle) |
| `dotnet` | `dotnet watch run` |

---

## Scaffolder Changes

A new file `internal/scaffolder/scaffolder_devcontainer.go` exposes:

```go
func GenerateDevContainerFiles(ctx registry.WeldContext, backendEntry registry.Entry) error
```

Called from the main scaffolder after existing file generation when `ctx.DevContainer == true`. Renders and writes the three templates. Rollback on error follows the same pattern as existing file writes.

`scaffolder.go` itself is not modified — the new function is additive.

---

## Detector Changes

`cmd/valla/main.go` adds `"docker"` to the runtime detection call. The boolean result is passed into `tui.New` so the model knows whether to include the "Fully Dockerized" option in the output structure list.

---

## Out of Scope

- Multi-service architecture (multiple frontends or backends) — deferred
- Non-Docker local dev environment generation
- CI/CD configuration generation
