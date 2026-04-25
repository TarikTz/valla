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

## Fully Dockerized Dev Environment

Valla's standout feature is the **Fully Dockerized** output mode. Instead of installing project dependencies (`node_modules`, Python virtual envs) directly on your machine, everything runs inside a Docker dev container. Your source code is the only thing on your host disk — packages never leave the container.

This matters for supply chain security. When a compromised package runs a malicious `postinstall` script, it executes inside the container and cannot reach your SSH keys, home directory, or system files. The blast radius is contained.

**What gets generated:**

```text
my-app/
├── frontend/
├── backend/
├── .devcontainer/
│   └── devcontainer.json   ← open in VS Code to enter the container
├── docker-compose.dev.yml  ← named volumes shadow node_modules / .venv
├── docker-compose.yml
├── Makefile
└── .env
```

Open the project folder in VS Code and click **Reopen in Container** — your entire dev environment starts inside Docker. Ports are forwarded automatically. The relevant language extension is pre-installed inside the container.

> **Requires Docker** — the option is only shown when Docker is detected on your machine.

## Quickstart

```bash
npx valla-cli
```

Or install globally:

```bash
npm install -g valla-cli
```

---

## Secure Serve — Zero-Config Local HTTPS

`valla serve` is a local TLS reverse proxy built into the CLI. It eliminates the friction of CORS mismatches and mixed-content browser warnings during full-stack development by giving every local service a real HTTPS URL with a trusted certificate.

### One-time setup (per machine)

```bash
# Generate a root CA and install it in the OS trust store
sudo npx valla-cli trust
```

After this step the browser shows a green padlock for every `*.valla.test` URL on this machine.

### Single service

```bash
# Proxy localhost:5500 → https://port5500.valla.test
valla serve 5500
```

If port 443 is taken the CLI falls back to 8443 then 9443 — the printed URL reflects the actual port.

### Multi-service routing

```bash
# Named subdomains via --map
valla serve --name film-portal --map "ui:3000,api:8080"
#  → https://ui.film-portal.test   (→ :3000)
#  → https://api.film-portal.test  (→ :8080)

# Port-range auto-mapping via --range
valla serve --name dev --range 5500-5502
#  → https://port5500.dev.test
#  → https://port5501.dev.test
#  → https://port5502.dev.test

# Combine both flags
valla serve --name myapp --map "web:3000" --range 8000-8001
```

Unknown subdomains receive a 502 with a plain-text list of configured routes.

### Declarative config (`valla.yaml`)

Running `valla serve` in a directory containing `valla.yaml` auto-loads the routing table. Explicit CLI flags override the file.

```yaml
# valla.yaml
project: film-portal
domain: test
services:
  - name: web
    port: 5500
    subdomain: preview   # https://preview.film-portal.test
  - name: api
    port: 8080
    subdomain: api       # https://api.film-portal.test
  - name: sso
    port: 8443
    subdomain: auth      # https://auth.film-portal.test
```

```bash
# Pick up valla.yaml automatically
cd film-portal && valla serve
```

### Interactive dashboard (`--ui`)

```bash
valla serve --name film-portal --map "ui:3000,api:8080" --ui
```

Renders a Bubbletea table view with live health checks, a rolling request log, and keyboard shortcuts:

```
  Valla Proxy Active  |  film-portal  |  .test
  ──────────────────────────────────────────────────────────────────────
  #    STATUS      SUBDOMAIN     TARGET              HTTPS URL
  ──────────────────────────────────────────────────────────────────────
  1    ● ONLINE    ui            localhost:3000       https://ui.film-portal.test:8443
  2    ● ONLINE    api           localhost:8080       https://api.film-portal.test:8443
  ──────────────────────────────────────────────────────────────────────
  [Recent requests]
  GET    ui            /css/style.css           200  12ms
  POST   api           /v1/search               201  45ms
  ──────────────────────────────────────────────────────────────────────
  Press 1-9 to open in browser · q to quit
```

Press a number key to open the corresponding URL in your default browser.

### Full flag reference

| Flag | Default | Description |
|------|---------|-------------|
| `[port]` | — | Single-port mode: proxy `localhost:<port>` |
| `--name` | `valla` | Subdomain namespace (`*.{name}.{domain}`) |
| `--domain` | `test` | TLD — use `test` or `localhost` (`.dev` triggers a confirmation) |
| `--map` | — | `"sub:port,sub2:port2"` named subdomain routing |
| `--range` | — | `start-end` port-range auto-mapping |
| `--ui` | off | Launch interactive Bubbletea dashboard |
| `--expose` | off | Bind to `0.0.0.0` instead of `127.0.0.1` for LAN sharing |

### TLD notes

| TLD | Suitable | Notes |
|-----|:--------:|-------|
| `.test` | ✓ | Reserved for local use; no `/etc/hosts` changes needed on most OS |
| `.localhost` | ✓ | Always resolves to 127.0.0.1 |
| `.lvh.me` | ✓ | Public wildcard DNS that resolves to 127.0.0.1 |
| `.dev` | ⚠ | Chrome/Edge enforce strict HSTS — the CLI prompts for confirmation |

---

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

- **Fully Dockerized** — dev environment runs entirely inside Docker; packages never touch your host machine
- Monorepo
- Separate folders
- Frontend only
- Backend only
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

### Fully Dockerized workflow

Requires [VS Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers).

```bash
npx valla-cli          # choose "Fully Dockerized" in the output structure step
cd my-app
code .                 # VS Code prompts "Reopen in Container" — click it
```

Inside the container, install dependencies and start your dev server:

```bash
# Node / Bun backend or frontend
npm install && npm run dev

# Python backend
pip install -r requirements.txt && uvicorn main:app --reload --host 0.0.0.0
```

All packages are installed inside the container. Your host machine only ever sees the source files.

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

### Fully Dockerized

```text
my-app/
    frontend/
    backend/
    .devcontainer/
        devcontainer.json
    .env
    docker-compose.yml
    docker-compose.dev.yml
    Makefile
```

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
- **Docker Desktop or Docker Engine** — required for Docker mode and Fully Dockerized dev environments
- **Internet access** — required for framework scaffolders and WordPress downloads
- **Go** — required only when running or building from source

## Roadmap

- More backend and database combinations
- Stronger end-to-end validation across stack combinations
- Non-interactive flags for CI and scripted usage
- Polished release automation for npm and GitHub Releases
- `valla serve --reload` — watch `valla.yaml` and rebuild routing table without restarting the listener

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

**Scaffolding**
- `internal/registry/` — framework metadata, embedded YAML, templates, Dockerfiles
- `internal/tui/` — interactive prompt flow
- `internal/wiring/` — `.env`, Docker Compose, HTTP client, and CORS wiring
- `internal/scaffolder/` — template rendering, file writes, rollback, rename handling
- `cmd/valla/` — CLI entrypoint and orchestration

**Secure Serve**
- `internal/proxy/` — ECDSA P-256 root CA lifecycle, SNI-based leaf cert cache, TLS reverse proxy engine, multi-service routing table, port-fallback `bindTLS`
- `internal/proxy/config/` — `valla.yaml` parsing and validation
- `internal/proxy/dashboard/` — Bubbletea TUI (health checks, request log, keyboard shortcuts), `WrapHandler` for request capture

New stack support is additive — adding a framework does not require changes to the core flow.

## Status

Available now on npm — `npx valla-cli`. Template coverage and generated output will continue to evolve as more stack combinations are exercised. See [Releases](https://github.com/tariktz/valla/releases) for the changelog.
