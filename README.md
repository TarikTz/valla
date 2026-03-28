# valla-cli

Stop assembling stacks by hand. `valla-cli` scaffolds frontend, backend, database, and Docker wiring in a single interactive flow — from zero to a running project in minutes.

## Quick Start

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

**npm (after first release is published):**

```bash
npx valla-cli
```

The repo includes a GoReleaser config (`.goreleaser.yaml`) for cross-platform binaries and an npm wrapper (`npm/`) that downloads the correct release artifact on first run.

> **Requirements:** Go is always needed. Add Node.js for frontend or Node-based backends. Add Docker Desktop or Docker Engine for Docker mode.

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

Install only what your selected stack needs.

- **Go** — required to run or build `valla-cli`
- **Node.js** — required for frontend scaffolds and Node-based backends
- **Docker Desktop or Docker Engine** — required for Docker mode
- **Internet access** — required for framework scaffolders and WordPress downloads

## Roadmap

- More backend and database combinations
- Stronger end-to-end validation across stack combinations
- Non-interactive flags for CI and scripted usage
- Polished release automation for npm and GitHub Releases

## Contributing

Contributions are welcome, especially in these areas:

- New framework or database entries
- Improvements to generated templates
- Cross-platform packaging and release setup
- End-to-end validation for more stack combinations
- UX improvements in the interactive flow

If you open a PR for a new stack, keep the change registry-driven and include tests for any new wiring behavior.

## Architecture

The implementation is organized around a registry of stack definitions and templates.

- `internal/registry/` — framework metadata, embedded YAML, templates, Dockerfiles
- `internal/tui/` — interactive prompt flow
- `internal/wiring/` — `.env`, Docker Compose, HTTP client, and CORS wiring
- `internal/scaffolder/` — template rendering, file writes, rollback, rename handling
- `cmd/valla/` — CLI entrypoint and orchestration

New stack support is additive — adding a framework does not require changes to the core flow.

## Status

Early-stage but usable today. Template coverage and generated output will continue to evolve as more stack combinations are exercised.
