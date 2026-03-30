# Valla Adoption Polish — Design Spec

**Date:** 2026-03-30
**Goal:** Grow adoption by improving first impressions, TUI polish, and maturity signals.
**Approach:** Balanced Adoption Stack (README → TUI polish → maturity signals)

---

## 1. README Overhaul

### Problem
The current README leads with an architecture description and contributing section. Visitors don't get a quick sense of the value prop, and there is no demo GIF showing the tool in action.

### Design

**Structure (top to bottom):**

1. **Logo/banner image** — an ASCII art or PNG banner with the `valla` name and tagline: *"Scaffold your full stack in seconds"*
2. **Badges row** — npm version, GitHub Actions CI status (only added after CI workflow has run successfully at least once on `main`), Go version, license
3. **Demo GIF** — recorded with `vhs` (Charm's tool), showing the full TUI flow in ~30 seconds: project name → stack selection → confirm → generated output. GIF is produced before the README is finalized (see implementation order).
4. **Rewritten intro** — 2–3 sentences max leading with the problem solved, not a feature list
5. **Quickstart** — `npx valla-cli` as line 1 of the install section, before any other method
6. **Supported stacks table** — existing table, lightly cleaned up
7. **What gets generated** — short directory tree examples for monorepo, separate, and WordPress modes

**What moves to the bottom:**
- Architecture overview
- Contributing guidelines
- Detailed internal implementation notes

### Files affected
- `README.md` — full rewrite of top half; bottom sections preserved but moved down
- `vhs/demo.tape` — new VHS script for recording the demo GIF (see VHS tape guidance below)
- `docs/demo.gif` — recorded output, embedded in README

### VHS tape guidance

The tape script should drive the following flow using VHS key sequences:
1. Type project name: `demo-app` → `Enter`
2. Select output structure: Monorepo → `Enter`
3. Select frontend runtime: Node → `Enter`
4. Select frontend framework: React → `Enter`
5. Select backend runtime: Go → `Enter`
6. Select backend framework: Gin → `Enter`
7. Select database: PostgreSQL → `Enter`
8. Skip ORM: None → `Enter`
9. Select env mode: Docker → `Enter`
10. Accept port defaults → `Enter`
11. Confirm → `Enter`

Total runtime target: ~25–35 seconds. Set VHS `Output` to `docs/demo.gif`.

---

## 2. TUI Polish

### Problem
The TUI is functional but plain. The first impression is a blank prompt with no brand presence. The success output is unstyled plain text.

### Design

**2a. Welcome banner**

The current `banner()` function in `internal/tui/styles.go` renders a single-line `⚡ valla-cli` header on every frame. Replace `banner()` with a multi-line ASCII art block rendered on every frame (same lifecycle as today — no architectural change needed). The new banner:

```
 ██╗   ██╗  █████╗  ██╗      ██╗       █████╗
 ██║   ██║ ██╔══██╗ ██║      ██║      ██╔══██╗
 ██║   ██║ ███████║ ██║      ██║      ███████║
 ╚██╗ ██╔╝ ██╔══██║ ██║      ██║      ██╔══██║
  ╚████╔╝  ██║  ██║ ███████╗ ███████╗ ██║  ██║
   ╚═══╝   ╚═╝  ╚═╝ ╚══════╝ ╚══════╝ ╚═╝  ╚═╝
  Scaffold your full stack in seconds.
```

Styled with a brand accent color (violet/purple) via lipgloss. The existing horizontal divider width in `styles.go` is updated to match the banner width (~52 characters).

If an update is available (see Section 3c), the update notice is appended as an additional line below the tagline within the banner string — no separate render path needed.

**2b. Stack summary card**

Rendered after the confirm step, before scaffolding begins. A lipgloss-bordered box. The "Mode" row is a combined display of two separate `WeldContext` fields: `OutputMode` (Monorepo/Separate/WordPress) and `EnvMode` (Local/.env or Docker). Implementers should concatenate them as `ctx.OutputMode + " · " + ctx.EnvMode`.

The summary card uses new field labels (`Frontend`, `Backend`, `Database`, `ORM`, `Mode`) and is rendered in `main.go` after the TUI exits. Do not reuse `summaryLines()` — that function feeds the in-TUI confirm step only and uses different labels (`Structure`, `Env mode`, etc.). The card in `main.go` is a separate render.

In all modes the project name appears in the card header as `valla · <project-name>`.

**Full-stack example:**
```
┌─────────────────────────────────────────┐
│  valla  ·  my-app                       │
│                                         │
│  Frontend    React (Bun)                │
│  Backend     Gin (Go)                   │
│  Database    PostgreSQL + Redis         │
│  ORM         Drizzle                    │
│  Mode        Monorepo · Docker          │
└─────────────────────────────────────────┘
```

**WordPress mode:** Show only Project, Mode (`WordPress · Docker`), and database port. Omit Frontend, Backend, ORM rows — consistent with the existing `summaryLines()` WordPress branch in `model.go`.

**Partial-stack modes (frontend-only or backend-only):** Omit the absent service row entirely. If no databases are selected, omit the Database and ORM rows.

**2c. Named spinner stages**

The current architecture runs scaffolding in `main.go` after `program.Run()` returns — the Bubble Tea program has already exited at this point. The named stages will be rendered **outside the TUI** using a simple stdout spinner (e.g. a lightweight loop printing to stdout with `\r` carriage return), not inside Bubble Tea. This keeps the scope simple and avoids restructuring scaffolding into a `tea.Cmd`.

Named stages, printed sequentially as each step completes in `main.go`:
1. `⠸ Scaffolding frontend...`
2. `⠸ Scaffolding backend...`
3. `⠸ Wiring environment...`
4. `⠸ Generating Docker config...` (only if `ctx.EnvMode == "docker"`)
5. `⠸ Injecting ORM config...` (only if ORM selected)

**Important:** The scaffold subprocess in `runScaffold()` currently sets `cmd.Stdout = os.Stdout` and `cmd.Stderr = os.Stderr`. While the stdout spinner is active, subprocess output will interleave with `\r`-based spinner updates and produce garbled output. Set `cmd.Stdout = nil` and `cmd.Stderr = nil` (or redirect to a buffer) during scaffolding, and surface stderr content only on non-zero exit.

The existing `phaseExecuting` / `SpinnerMsg` in the TUI is dead code from a planned feature. It is left untouched (not removed, not used) — that refactor is out of scope.

**2d. Styled success output**

Replaces the current plain-text next-steps with:

- Large green `✓  Done!` header rendered via lipgloss
- Generated directory tree (styled in muted color)
- Next-step commands in a highlighted block (e.g. `cd my-app && docker compose up`)

Rendered to stdout after scaffolding completes in `main.go`, not inside the TUI.

### Files affected
- `internal/tui/styles.go` — replace `banner()` with multi-line ASCII art version, update divider width, add card/success style definitions
- `internal/tui/model.go` — update `View()` to render update-available notice within banner string if `m.updateNotice != ""`; add `UpdateAvailableMsg` handling in `Update()`
- `cmd/valla/main.go` — render stack summary card after confirm, emit named stdout stages during scaffolding, render styled success output

---

## 3. Maturity Signals

### Problem
No `--version` flag, no CI badge, no update checker. These are low-effort signals that separate "hobby project" from "maintained tool."

### Design

**3a. `--version` flag**

```
valla-cli --version
# → valla-cli v0.4.1
```

A package-level `var version = "dev"` in `cmd/valla/main.go` is set at compile time via `-ldflags`.

The current `.github/workflows/release.yml` builds binaries with plain `go build` — it does **not** use goreleaser. Add `-ldflags "-X main.version=${{ github.ref_name }}"` to each `go build` line in the release workflow:

```yaml
go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_darwin_arm64 ./cmd/valla
# (repeat for all 5 platforms)
```

Note: `.goreleaser.yaml` exists in the repo but is not wired into the release process. Do not modify it as part of this work.

The `--version` flag in `main.go` reads `version` and prints `valla-cli <version>`.

**3b. CI workflow**

New file: `.github/workflows/ci.yml`

Triggers on: push to `main`, all pull requests.

Steps:
1. `actions/checkout`
2. `actions/setup-go` with Go version matching `go.mod`
3. `go test ./...`
4. `go build ./...`

The CI badge is added to the README **only after** the workflow has run successfully on `main` at least once — badge will show a 404/error before that first run. See implementation order.

**3c. Update checker**

On startup, before `tea.NewProgram` is called, spawn a goroutine that checks the GitHub releases API:

```
GET https://api.github.com/repos/tariktz/valla/releases/latest
```

The goroutine writes its result into a channel. After `tea.NewProgram` is created but before `program.Run()` is called, pass the channel (or a pointer to a string) into the `Model` so that the TUI can receive the update notice.

The correct approach: launch the goroutine **after** `tea.NewProgram(model)` assigns the `program` variable, capturing it by closure. Call `program.Send(UpdateAvailableMsg{version: "v0.x.x"})` from the goroutine — this is safe because `Send` blocks until the program's event loop is ready. Launching before `tea.NewProgram` would create a race. This requires:

- `UpdateAvailableMsg` defined in `internal/tui/model.go`
- A `m.updateNotice string` field on `Model`
- `Update()` handles `UpdateAvailableMsg` by setting `m.updateNotice`
- `View()` appends `m.updateNotice` to the banner string when non-empty (e.g. `"\n  ⚡ Update available: v0.x.x  →  npm install -g valla-cli"`)

Respects `VALLA_NO_UPDATE_CHECK=1` environment variable to disable. No `--no-update-check` flag — env var is sufficient and avoids a redundant flag entry in `--help`.

Fails silently on network errors. Timeout: 3 seconds.

**3d. `--help` cleanup**

Replace the default Go flag usage output with a custom `flag.Usage` function that:
- Shows the valla name/tagline
- Lists available flags: `--version`, `--help`
- Notes: `VALLA_NO_UPDATE_CHECK=1` env var to disable update checks
- Shows "run `valla-cli` with no flags to start the interactive wizard"

### Files affected
- `cmd/valla/main.go` — `--version` flag, custom `flag.Usage`, update checker goroutine with `program.Send`
- `internal/tui/model.go` — `UpdateAvailableMsg` type definition, `updateNotice string` field, `Update()` handler, `View()` banner integration
- `.github/workflows/release.yml` — add `-ldflags "-X main.version=${{ github.ref_name }}"` to each `go build` line
- `.github/workflows/ci.yml` — new file

---

## Implementation Order

1. **CI workflow** (`.github/workflows/ci.yml`) — unblocks the CI badge; push to `main` and confirm first run succeeds before continuing
2. **`--version` flag + ldflags in `release.yml`** — quick win, high signal
3. **Welcome banner + styles** — replace `banner()`, update divider width, add style constants
4. **Stack summary card** — builds on banner styles, rendered in `main.go` before scaffolding
5. **Named stdout spinner stages** — wire into scaffolding steps in `main.go`
6. **Styled success output** — final TUI polish
7. **`UpdateAvailableMsg` + model field** — add message type and handler to `model.go`
8. **Update checker goroutine** — add to `main.go`, integrates with step 7
9. **`--help` cleanup** — done after final flag set is known
10. **VHS tape script** (`vhs/demo.tape`) — record demo GIF against the polished TUI
11. **README overhaul** — done last; embeds GIF from step 10, adds CI badge (safe now that CI has run)

---

## Non-Goals

- Landing page / docs site (deferred until usage grows)
- Shell completions (not in scope for this iteration)
- Telemetry / analytics
- Plugin system
- Moving scaffolding inside the Bubble Tea program as a `tea.Cmd` (deferred)
