# Getting started

Valla scaffolds a complete full-stack project in one terminal flow. No boilerplate hunting, no manual wiring.

## Installation

::: code-group

```bash [npx (no install)]
npx valla-cli
```

```bash [npm]
npm install -g valla-cli
valla
```

```bash [Homebrew]
# Coming soon
brew install valla-cli
valla
```

```bash [Binary]
# Download a pre-built binary for your platform from:
# https://github.com/tariktz/valla/releases
./valla
```

:::

> The npm package and pre-built binaries bundle the Go binary — **Go is not required** on your machine.

## How it works

The CLI walks you through nine prompts and generates a ready-to-run project:

1. Project name
2. Output structure (monorepo, separate folders, Fully Dockerized, frontend-only, backend-only, WordPress)
3. Frontend runtime and framework
4. Backend runtime and framework
5. Database — multi-select, combine any combination
6. ORM (shown for eligible Node.js stacks with a SQL database)
7. Environment wiring (local `.env` or Docker Compose)
8. Optional port overrides
9. Confirmation and generation

## Requirements

> When using `npx valla-cli` or a pre-built binary, Go is **not** required. The npm wrapper downloads the correct binary for your platform automatically.

Install only what your selected stack needs:

| Dependency | When needed |
|---|---|
| **Node.js** | Frontend scaffolds, Node-based backends |
| **Docker Desktop / Engine** | Docker mode, Fully Dockerized dev environments |
| **Internet access** | Framework scaffolders, WordPress downloads |
| **Go** | Building or running from source only |

## Supported platforms

| Platform | amd64 | arm64 |
|---|:---:|:---:|
| macOS | ✓ | ✓ |
| Linux | ✓ | ✓ |
| Windows | ✓ | ✓ |
