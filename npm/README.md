# valla-cli

Stop assembling stacks by hand. `valla-cli` scaffolds frontend, backend, database, and Docker wiring in a single interactive flow — from zero to a running project in minutes.

Open source — contributions welcome at [github.com/tariktz/valla](https://github.com/tariktz/valla).

## Usage

```bash
npx valla-cli
```

Or install globally:

```bash
npm install -g valla-cli
valla-cli
```

On first run, the correct pre-built binary for your platform is downloaded and cached automatically. **No Go installation required.**

- macOS/Linux: `~/.cache/valla-cli/`
- Windows: `%LOCALAPPDATA%\valla-cli\`

## What It Does

- Scaffolds frontend and backend together in a single interactive terminal flow
- Generates `.env` for local development
- Generates `docker-compose.yml` for containerized development
- Wires frontend API defaults to the selected backend port automatically
- Injects backend CORS configuration for supported backends
- Supports monorepo and separate-folder output modes
- Includes a dedicated WordPress mode with Docker services and a starter theme

## How It Works

The CLI walks through a short set of prompts:

1. Project name
2. Output structure (monorepo, separate folders, or WordPress)
3. Frontend runtime and framework
4. Backend runtime and framework
5. Database
6. Local `.env` or Docker Compose
7. Optional port overrides
8. Confirmation and scaffolding

For WordPress, the CLI downloads the latest WordPress source, prepares Docker services, and creates a starter theme.

## Supported Stacks

**Frontend:** React, Vue, Angular, Svelte, Astro

**Backend:** Go (Gin), Go (Fiber), Go boilerplate, Node.js (Express), Node.js (Nest), Node.js boilerplate

**Database:** PostgreSQL, SQLite

**Output modes:** Monorepo, Separate folders, WordPress

## Generated Project Shapes

**Monorepo:**
```
my-app/
├── frontend/
├── backend/
├── .env
└── docker-compose.yml
```

**Separate folders:**
```
my-app/
├── my-app-frontend/
├── my-app-backend/
├── .env
└── docker-compose.yml
```

**WordPress:**
```
my-wordpress-project/
├── .env
├── docker-compose.yml
└── wordpress/
    └── wp-content/
        └── themes/
            └── my-wordpress-project/
```

## Supported Platforms

| OS      | x64 | arm64 |
|---------|:---:|:-----:|
| macOS   |  ✓  |   ✓   |
| Linux   |  ✓  |   ✓   |
| Windows |  ✓  |   ✓   |

## Requirements

> Go is **not** required — the npm wrapper downloads the correct binary automatically.

Install only what your selected stack needs:

- **Node.js** — required for frontend scaffolds and Node-based backends
- **Docker Desktop or Docker Engine** — required for Docker mode
- **Internet access** — required for framework scaffolders and WordPress downloads

## More

Full documentation, architecture details, and contribution guide: [github.com/tariktz/valla](https://github.com/tariktz/valla)
