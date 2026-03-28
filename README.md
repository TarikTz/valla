# valla-cli

Interactive full-stack project scaffolding for teams that want a usable starting point in minutes instead of assembling frontend, backend, env wiring, and Docker by hand.

`valla-cli` is a Go CLI with a Bubble Tea TUI. It scaffolds a frontend, a backend, and optional database wiring, then generates the local `.env` or `docker-compose.yml` needed to run the stack. It also supports a dedicated WordPress mode for local theme development with a bind-mounted WordPress source tree.

## Status

This project is usable today, but it is still early-stage. Template coverage and generated output will continue to evolve as more stack combinations are exercised.

## What It Does

- Scaffolds full-stack projects through an interactive terminal flow
- Supports monorepo and separate-folder output modes
- Supports a WordPress preset with local files mounted into Docker
- Generates `.env` for local development
- Generates `docker-compose.yml` for containerized development
- Wires frontend API defaults to the selected backend port
- Injects backend CORS configuration for supported backends
- Uses a registry-driven architecture so new frameworks can be added without rewriting the core flow

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

For standard app stacks, the result is a generated frontend and backend, plus environment wiring. For WordPress, the CLI downloads the latest WordPress source, prepares Docker services, and creates a starter theme inside `wordpress/wp-content/themes/<project-slug>`.

## Installation

### Run from source

```bash
git clone https://github.com/tariktz/valla-cli
cd valla-cli
go run ./cmd/valla
```

To build a local binary:

```bash
go build -o valla-cli ./cmd/valla
./valla-cli
```

### npm (after first release is published)

```bash
npx valla-cli
```

### Distribution

The repo includes a GoReleaser config (`.goreleaser.yaml`) for cross-platform binaries and an npm wrapper (`npm/`) that downloads the correct release artifact on first run.

## Requirements

Install only what your selected stack needs.

- Go for running or building `valla-cli`
- Node.js for frontend scaffolds and Node-based backends
- Docker Desktop or Docker Engine if you choose Docker mode
- Internet access for framework scaffolders and WordPress downloads

## Usage

Run the CLI:

```bash
go run ./cmd/valla
```

The current interface is fully interactive. There is no stable flag-based or `--help` workflow yet.

### Typical local workflow

```bash
go run ./cmd/valla
cd my-app
```

Then:

- in the frontend: install dependencies and start the dev server
- in the backend: run the server for the selected stack

### Typical Docker workflow

```bash
go run ./cmd/valla
cd my-app
docker-compose up -d
```

### WordPress workflow

Choose `WordPress` in the output structure step. The generator will create:

- `.env`
- `docker-compose.yml`
- `wordpress/`
- `wordpress/wp-content/themes/<project-slug>/`

After generation:

```bash
cd my-wordpress-project
docker-compose up -d
```

Then open `http://localhost:<wordpress-port>` and finish the WordPress setup in the browser.

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

## Development

Build the project:

```bash
go build ./...
```

Run the internal test suite:

```bash
go test ./internal/...
```

## Architecture

The implementation is organized around a registry of stack definitions and templates.

- `internal/registry/`: framework metadata, embedded YAML, templates, dockerfiles
- `internal/tui/`: interactive prompt flow
- `internal/wiring/`: `.env`, Docker Compose, HTTP client, and CORS wiring
- `internal/scaffolder/`: template rendering, file writes, rollback, rename handling
- `cmd/valla/`: CLI entrypoint and orchestration

This structure is meant to keep new stack support additive rather than invasive.

## Contributing

Contributions are welcome, especially in these areas:

- new framework or database entries
- improvements to generated templates
- cross-platform packaging and release setup
- end-to-end validation for more stack combinations
- UX improvements in the interactive flow

If you open a PR for a new stack, keep the change registry-driven and include tests for any new wiring behavior.

## Roadmap

- more backend and database combinations
- stronger end-to-end validation across stack combinations
- non-interactive flags for CI and scripted usage
- polished release automation for npm and GitHub Releases