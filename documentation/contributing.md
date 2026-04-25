# Contributing

valla-cli is open source and contributions are very welcome.

## Getting started

```bash
git clone https://github.com/tariktz/valla
cd valla
go test ./...
```

Open an [issue](https://github.com/tariktz/valla/issues) to discuss ideas before opening large PRs.

## High-impact areas

| Area | Where to look |
|---|---|
| New backends or frameworks | Add a registry YAML entry — no core changes needed for scaffolding; wiring patches require a registry entry update |
| New database support | Extend the registry and wiring engine |
| Generated template improvements | `internal/registry/data/post-scaffold/` |
| Cross-platform packaging | GoReleaser config and npm wrapper in `npm/` |
| End-to-end validation | `internal/wiring/` — test more stack combinations |
| UX improvements | `internal/tui/` — the interactive prompt flow |

If you open a PR for a new stack, keep the change registry-driven and include tests for any new wiring behavior.

## Architecture

**Scaffolding**

| Package | Responsibility |
|---|---|
| `internal/registry/` | Framework metadata, embedded YAML, templates, Dockerfiles |
| `internal/tui/` | Interactive prompt flow (Bubbletea) |
| `internal/wiring/` | `.env`, Docker Compose, HTTP client, and CORS wiring |
| `internal/scaffolder/` | Template rendering, file writes, rollback, rename handling |
| `cmd/valla/` | CLI entrypoint and orchestration |

**Secure Serve**

| Package | Responsibility |
|---|---|
| `internal/proxy/` | ECDSA P-256 root CA lifecycle, SNI-based leaf cert cache, TLS reverse proxy, multi-service routing, port-fallback bind |
| `internal/proxy/config/` | `valla.yaml` parsing and validation |
| `internal/proxy/dashboard/` | Bubbletea TUI (health checks, request log, keyboard shortcuts), `WrapHandler` for request capture |

New stack support is additive — adding a framework does not require changes to the core flow.

## Running tests

```bash
go test ./...
```
