# valla-cli

[![npm version](https://img.shields.io/npm/v/valla-cli.svg)](https://www.npmjs.com/package/valla-cli)
[![npm downloads](https://img.shields.io/npm/dm/valla-cli.svg)](https://www.npmjs.com/package/valla-cli)
[![GitHub release](https://img.shields.io/github/v/release/tariktz/valla)](https://github.com/tariktz/valla/releases)
[![License: MIT](https://img.shields.io/github/license/tariktz/valla)](https://github.com/tariktz/valla/blob/main/LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/tariktz/valla)](go.mod)

Stop assembling stacks by hand. `valla-cli` scaffolds frontend, backend, database, and Docker wiring in a single interactive flow — from zero to a running project in minutes.

valla-cli is an **open source** project. Contributions are welcome — see [Contributing](#contributing) below.

## Quick Start

**npm (recommended):**

```bash
npx valla-cli
```

Or install globally:

```bash
npm install -g valla-cli
valla-cli
```

On first run, the correct pre-built binary for your platform is downloaded and cached automatically. No Go installation needed when using npm.

**Run from source:**

```bash
git clone https://github.com/tariktz/valla
cd valla
go run ./cmd/valla
```

**Build a local binary:**

```bash
go build -o valla-cli ./cmd/valla
./valla-cli
```

## What It Does

- Scaffolds frontend and backend together in a single interactive terminal flow
- Generates `.env` for local development
- Generates `docker-compose.yml` for containerized development
- Wires frontend API defaults to the selected backend port automatically
- Injects backend CORS configuration for supported backends
- Supports monorepo and separate-folder output modes
- Includes a dedicated WordPress mode with Docker services and a starter theme

## Supported Stacks

### Frontend

- React
- Vue
- Angular
- Svelte
- Astro

### Backend

- Go (Gin)
- Go (Fiber)
- Go boilerplate template
- Node.js (Express)
- Node.js (Nest)
- Node.js boilerplate template

### Database

- PostgreSQL
- SQLite

### Output Modes

- Monorepo
- Separate folders
- WordPress

## Supported Platforms

| Platform | amd64 | arm64 |
|----------|:-----:|:-----:|
| macOS    |   ✓   |   ✓   |
| Linux    |   ✓   |   ✓   |
| Windows  |   ✓   |   ✓   |

## How It Works

The CLI walks through a short set of prompts:

1. Project name
2. Output structure
3. Frontend runtime and framework
4. Backend runtime and framework
5. Database
6. Local `.env` or Docker Compose
7. Optional port overrides
8. Confirmation and scaffolding

For standard stacks, the result is a generated frontend and backend with environment wiring. For WordPress, the CLI downloads the latest WordPress source, prepares Docker services, and creates a starter theme inside `wordpress/wp-content/themes/<project-slug>`.

## Usage

### Local workflow

```bash
go run ./cmd/valla
cd my-app
```

Then start each piece:

```bash
# Frontend
cd frontend && npm install && npm run dev

# Backend — command depends on your selected stack
```

### Docker workflow

```bash
go run ./cmd/valla
cd my-app
docker-compose up -d
```

### WordPress workflow

Choose `WordPress` in the output structure step. After generation:

```bash
cd my-wordpress-project
docker-compose up -d
```

Then open `http://localhost:<wordpress-port>` and complete the WordPress setup in the browser.

## Generated Project Shapes

### Monorepo

```text
my-app/
    frontend/
    backend/
    .env
    docker-compose.yml
```

### Separate folders

```text
my-app/
    my-app-frontend/
    my-app-backend/
    .env
    docker-compose.yml
```

### WordPress

```text
my-wordpress-project/
    .env
    docker-compose.yml
    wordpress/
        wp-content/
            themes/
                my-wordpress-project/
```

## Requirements

> **Note:** When using `npx valla-cli` or a pre-built binary, Go is **not** required. The npm wrapper downloads the correct binary for your platform automatically.

Install only what your selected stack needs:

- **Node.js** — required for frontend scaffolds and Node-based backends
- **Docker Desktop or Docker Engine** — required for Docker mode
- **Internet access** — required for framework scaffolders and WordPress downloads
- **Go** — required only when running or building from source

## Roadmap

- .NET backends: ASP.NET Core Web API and .NET Minimal API
- Java backends: Spring Boot (Maven/Gradle) and Quarkus (Maven/Gradle)
- More backend and database combinations
- Stronger end-to-end validation across stack combinations
- Non-interactive flags for CI and scripted usage
- Polished release automation for npm and GitHub Releases

## Contributing

valla-cli is an open source project and contributions are very welcome!

The most impactful areas to contribute:

- **New backends or frameworks** — add a registry YAML entry (no core changes needed for scaffolding; wiring patches require a registry entry update)
- **New database support** — extend the registry and wiring engine
- **Improvements to generated templates** — post-scaffold source files live in `internal/registry/data/post-scaffold/`
- **Cross-platform packaging and release setup** — GoReleaser config and npm wrapper in `npm/`
- **End-to-end validation** — test more stack combinations in `internal/wiring/`
- **UX improvements** — the interactive TUI flow lives in `internal/tui/`

If you open a PR for a new stack, keep the change registry-driven and include tests for any new wiring behavior.

**Get started:**

```bash
git clone https://github.com/tariktz/valla
cd valla
go test ./...
```

Open an [issue](https://github.com/tariktz/valla/issues) to discuss ideas before opening large PRs.

## Architecture

The implementation is organized around a registry of stack definitions and templates.

- `internal/registry/` — framework metadata, embedded YAML, templates, Dockerfiles
- `internal/tui/` — interactive prompt flow
- `internal/wiring/` — `.env`, Docker Compose, HTTP client, and CORS wiring
- `internal/scaffolder/` — template rendering, file writes, rollback, rename handling
- `cmd/valla/` — CLI entrypoint and orchestration

New stack support is additive — adding a framework does not require changes to the core flow.

## Status

Available now on npm — `npx valla-cli`. Template coverage and generated output will continue to evolve as more stack combinations are exercised. See [Releases](https://github.com/tariktz/valla/releases) for the changelog.
