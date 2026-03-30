<div align="center">

```
 ██╗   ██╗  █████╗  ██╗      ██╗       █████╗
 ██║   ██║ ██╔══██╗ ██║      ██║      ██╔══██╗
 ██║   ██║ ███████║ ██║      ██║      ███████║
 ╚██╗ ██╔╝ ██╔══██║ ██║      ██║      ██╔══██║
  ╚████╔╝  ██║  ██║ ███████╗ ███████╗ ██║  ██║
   ╚═══╝   ╚═╝  ╚═╝ ╚══════╝ ╚══════╝ ╚═╝  ╚═╝
```

**Scaffold your full stack in seconds.**

[![npm version](https://img.shields.io/npm/v/valla-cli)](https://www.npmjs.com/package/valla-cli)
[![CI](https://github.com/tariktz/valla/actions/workflows/ci.yml/badge.svg)](https://github.com/tariktz/valla/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/tariktz/valla)](go.mod)
[![License](https://img.shields.io/github/license/tariktz/valla)](LICENSE)

![valla demo](demo.gif)

</div>

---

Stop wiring up frontend, backend, database, and Docker by hand. Valla scaffolds your entire stack in one terminal flow — pick your frameworks, hit Enter, and get a production-ready project structure with environment config and Docker Compose included.

## Quickstart

```bash
npx valla-cli
```

Or install globally:

```bash
npm install -g valla-cli
```

## Supported Stacks

### Frontend

| Framework | Node | Bun | Server-side |
|-----------|:----:|:---:|:-----------:|
| React | ✓ | ✓ | |
| Vue | ✓ | ✓ | |
| Angular | ✓ | ✓ | |
| Svelte (SvelteKit) | ✓ | ✓ | ✓ |
| Astro | ✓ | ✓ | ✓ |
| Next.js | ✓ | ✓ | ✓ |
| TanStack Start | ✓ | ✓ | ✓ |

> Server-side frameworks are eligible for ORM integration in frontend-only mode.

### Backend

| Language | Framework / Template |
|----------|----------------------|
| Go | Gin |
| Go | Fiber |
| Go | Boilerplate |
| Node.js | Express |
| Node.js | NestJS |
| Node.js | Boilerplate |
| Python | FastAPI |
| Python | Flask |
| Python | Django |
| .NET | ASP.NET Core (Web API) |
| .NET | Minimal API |
| Java | Spring Boot (Maven) |
| Java | Spring Boot (Gradle) |
| Java | Quarkus (Maven) |
| Java | Quarkus (Gradle) |

### Database

Multiple databases can be selected simultaneously. SQLite and None are exclusive options.

| Database | Docker service | Combinable |
|----------|:--------------:|:----------:|
| PostgreSQL | ✓ | ✓ |
| MySQL | ✓ | ✓ |
| MariaDB | ✓ | ✓ |
| MongoDB | ✓ | ✓ |
| Redis | ✓ | ✓ |
| SQLite | | exclusive |

### ORM _(optional)_

Shown when at least one SQL database is selected and the project uses a Node.js runtime (Node backend or a server-side frontend framework).

| ORM | Generated files |
|-----|------------------|
| Prisma | `prisma/schema.prisma`, `prisma.config.ts` |
| Drizzle | `drizzle.config.ts`, `src/db/index.ts` |

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
5. Database (multi-select — combine PostgreSQL, MySQL, MariaDB, MongoDB, Redis, or pick SQLite/None)
6. ORM selection (Prisma, Drizzle, or None — shown only for eligible stacks)
7. Local `.env` or Docker Compose
8. Optional port overrides
9. Confirmation and scaffolding

For standard stacks, the result is a generated frontend and backend with environment wiring, a composed `.env`, and optionally ORM config files. For WordPress, the CLI downloads the latest WordPress source, prepares Docker services, and creates a starter theme inside `wordpress/wp-content/themes/<project-slug>`.

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

With Prisma selected:

```text
my-app/
    frontend/
    backend/
        prisma/
            schema.prisma
        prisma.config.ts
    .env               ← includes DATABASE_URL
    docker-compose.yml
```

### Separate folders

```text
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
