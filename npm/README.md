# valla-cli

Stop assembling stacks by hand. `valla-cli` scaffolds frontend, backend, database, and Docker wiring in a single interactive flow — from zero to a running project in minutes.

## Usage

```bash
npx valla-cli
```

Or install globally:

```bash
npm install -g valla-cli
valla-cli
```

On first run, the correct binary for your platform is downloaded and cached:
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

## Supported Stacks

**Frontend:** React, Vue, Angular, Svelte, Astro

**Backend:** Go (Gin), Go (Fiber), Go boilerplate, Node.js (Express), Node.js (Nest), Node.js boilerplate

**Database:** PostgreSQL, SQLite

**Output modes:** Monorepo, Separate folders, WordPress

## Supported Platforms

| OS      | x64 | arm64 |
|---------|-----|-------|
| macOS   | ✓   | ✓     |
| Linux   | ✓   | ✓     |
| Windows | ✓   | ✓     |

## Requirements

Install only what your selected stack needs:

- **Node.js** — required for frontend scaffolds and Node-based backends
- **Docker Desktop or Docker Engine** — required for Docker mode
- **Internet access** — required for framework scaffolders and WordPress downloads

## More

Full documentation, architecture details, and contribution guide: [github.com/tariktz/valla-cli](https://github.com/tariktz/valla-cli)
