# Valla Adoption Polish Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve first impressions, TUI polish, and maturity signals to grow adoption of the valla-cli tool.

**Architecture:** Three independent layers implemented in order: (1) infrastructure maturity (CI, --version), (2) TUI polish (banner, summary card, spinner, success output, update checker), (3) README overhaul with demo GIF. All TUI changes use the existing lipgloss/bubbles dependency tree — no new Go deps. The update checker uses `program.Send` from a goroutine launched after `tea.NewProgram` to avoid races.

**Tech Stack:** Go 1.25, charmbracelet/bubbletea, charmbracelet/lipgloss, GitHub Actions, VHS (Charm's terminal recorder)

**Spec:** `docs/superpowers/specs/2026-03-30-valla-adoption-polish-design.md`

---

## Chunk 1: Infrastructure & Maturity Signals

### Task 1: CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create the CI workflow file**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Test
        run: go test ./...

      - name: Build
        run: go build ./...
```

- [ ] **Step 2: Verify the file is valid**

Run: `cat .github/workflows/ci.yml`
Expected: file contents shown, no syntax errors

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add CI workflow for tests and build on push/PR"
```

> **Note:** Push to `main` to trigger the first CI run. The CI badge must NOT be added to the README until the workflow has successfully run once on `main` — the badge URL will 404 before that.

---

### Task 2: `--version` flag and release ldflags

**Files:**
- Modify: `cmd/valla/main.go` (top of file, before `func main()`)
- Modify: `.github/workflows/release.yml` (Build binaries step)

- [ ] **Step 1: Add `version` variable and `--version` flag to `main.go`**

Add directly after the `import` block, before `func main()`:

```go
var version = "dev"
```

At the very start of `func main()`, before the `entries, err := registry.Load()` line, add:

```go
if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
    fmt.Printf("valla-cli %s\n", version)
    return
}
```

- [ ] **Step 2: Verify it compiles and works locally**

Run: `go run ./cmd/valla --version`
Expected: `valla-cli dev`

- [ ] **Step 3: Add ldflags to each `go build` line in `release.yml`**

Replace the "Build binaries" step content in `.github/workflows/release.yml`:

```yaml
      - name: Build binaries
        run: |
          mkdir dist
          GOOS=darwin  GOARCH=arm64 go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_darwin_arm64     ./cmd/valla
          GOOS=darwin  GOARCH=amd64 go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_darwin_amd64     ./cmd/valla
          GOOS=linux   GOARCH=arm64 go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_linux_arm64      ./cmd/valla
          GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_linux_amd64      ./cmd/valla
          GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=${{ github.ref_name }}" -o dist/valla-cli_windows_amd64.exe ./cmd/valla
```

- [ ] **Step 4: Run existing tests to confirm nothing is broken**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/valla/main.go .github/workflows/release.yml
git commit -m "feat: add --version flag with ldflags injection for releases"
```

---

## Chunk 2: TUI Polish — Styles and Model

### Task 3: Replace banner and add style definitions

**Files:**
- Modify: `internal/tui/styles.go`

The current `banner()` renders a single-line title + 60-char divider. Replace it with a multi-line ASCII art block. The divider is also used in step rendering — update its width to 52 chars to match the new banner width.

- [ ] **Step 1: Replace `internal/tui/styles.go` entirely**

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleBannerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	styleBannerTagline = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	styleBannerDivider = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#333333"))

	styleUpdateNotice = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	// styleCardBorder is used by the stack summary card rendered in main.go.
	StyleCardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	// StyleCardLabel is the dim left-side label in a card row.
	StyleCardLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Width(12)

	// StyleCardValue is the bright right-side value in a card row.
	StyleCardValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0"))

	// StyleCardHeader is the "valla · <project>" header line.
	StyleCardHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	// StyleSuccessHeader is the "✓  Done!" line.
	StyleSuccessHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#10B981"))

	// StyleSuccessTree is the muted directory tree.
	StyleSuccessTree = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	// StyleSuccessCmd is the highlighted next-step commands.
	StyleSuccessCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)
)

const bannerASCII = ` ██╗   ██╗  █████╗  ██╗      ██╗       █████╗
 ██║   ██║ ██╔══██╗ ██║      ██║      ██╔══██╗
 ██║   ██║ ███████║ ██║      ██║      ███████║
 ╚██╗ ██╔╝ ██╔══██║ ██║      ██║      ██╔══██║
  ╚████╔╝  ██║  ██║ ███████╗ ███████╗ ██║  ██║
   ╚═══╝   ╚═╝  ╚═╝ ╚══════╝ ╚══════╝ ╚═╝  ╚═╝`

func banner(updateNotice string) string {
	art := styleBannerTitle.Render(bannerASCII)
	tagline := styleBannerTagline.Render("  Scaffold your full stack in seconds.")
	divider := styleBannerDivider.Render("────────────────────────────────────────────────────")
	result := art + "\n" + tagline
	if updateNotice != "" {
		result += "\n" + styleUpdateNotice.Render(updateNotice)
	}
	result += "\n" + divider
	return result
}
```

> **⚠️ Atomic edit required:** Steps 1 and 2 must be completed before running any build or test. `styles.go` changes `banner()` to `banner(updateNotice string)` — the project will not compile until `model.go` is also updated in Step 2.

- [ ] **Step 2: Update the `banner()` call in `model.go`**

`model.go` line 432 calls `banner()` with no arguments. Update it to pass `m.updateNotice`:

Old:
```go
return banner() + "\n\n" + m.current.View()
```

New:
```go
return banner(m.updateNotice) + "\n\n" + m.current.View()
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Run tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/styles.go internal/tui/model.go
git commit -m "feat: replace banner with multi-line ASCII art and add card/success styles"
```

---

### Task 4: Add `UpdateAvailableMsg` to model

**Files:**
- Modify: `internal/tui/model.go`

- [ ] **Step 1: Write a test for `UpdateAvailableMsg` handling**

In `internal/tui/model_test.go` (create if it doesn't exist), add:

```go
package tui

import (
	"testing"
)

func TestUpdateAvailableMsgSetsNotice(t *testing.T) {
	m := Model{}
	updatedModel, _ := m.Update(UpdateAvailableMsg{Version: "v9.9.9"})
	got := updatedModel.(Model).updateNotice
	want := "  ⚡ Update available: v9.9.9  →  npm install -g valla-cli"
	if got != want {
		t.Errorf("updateNotice = %q, want %q", got, want)
	}
}

func TestUpdateAvailableMsgEmpty(t *testing.T) {
	m := Model{}
	updatedModel, _ := m.Update(UpdateAvailableMsg{Version: ""})
	got := updatedModel.(Model).updateNotice
	if got != "" {
		t.Errorf("expected empty updateNotice for empty version, got %q", got)
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./internal/tui/... -run TestUpdateAvailable -v`
Expected: compile error — `UpdateAvailableMsg` undefined

- [ ] **Step 3: Add `UpdateAvailableMsg`, `updateNotice` field, and `Update()` handler**

In `internal/tui/model.go`:

Add the type before the `Model` struct:
```go
// UpdateAvailableMsg is sent by the update-checker goroutine in main.go
// when a newer version of valla-cli is available on GitHub.
type UpdateAvailableMsg struct {
	Version string
}
```

Add `updateNotice string` to the `Model` struct:
```go
type Model struct {
	phase   phase
	current tea.Model
	ctx     registry.WeldContext
	entries []registry.Entry

	feRuntimeOpts []steps.RuntimeOption
	beRuntimeOpts []steps.RuntimeOption

	selectedFERuntime string
	selectedBERuntime string

	confirmed    bool
	updateNotice string
}
```

Add a case for `UpdateAvailableMsg` in `Update()`, inside the top-level switch, after the `ErrMsg` case:
```go
case UpdateAvailableMsg:
    if msg.Version != "" {
        m.updateNotice = fmt.Sprintf("  ⚡ Update available: %s  →  npm install -g valla-cli", msg.Version)
    }
    return m, nil
```

- [ ] **Step 4: Run the test to confirm it passes**

Run: `go test ./internal/tui/... -run TestUpdateAvailable -v`
Expected: PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat: add UpdateAvailableMsg and updateNotice banner integration"
```

---

## Chunk 3: main.go Polish

### Task 5: Stack summary card

**Files:**
- Modify: `cmd/valla/main.go`

The summary card is rendered in `main.go` after the TUI exits (after `tuiModel.Context()`) and before scaffolding begins. It uses the exported `Style*` vars from `internal/tui/styles.go`.

- [ ] **Step 1: Write a test for `renderSummaryCard`**

Create `cmd/valla/main_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
)

func TestRenderSummaryCardFullStack(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName: "my-app",
		FrontendID:  "react-node",
		BackendID:   "gin",
		DatabaseIDs: []string{"postgres"},
		ORMID:       "drizzle",
		OutputMode:  "monorepo",
		EnvMode:     "docker",
	}
	entries := []registry.Entry{
		{ID: "react-node", Name: "React", Runtime: "node", Type: "frontend"},
		{ID: "gin", Name: "Gin", Runtime: "go", Type: "backend"},
		{ID: "postgres", Name: "PostgreSQL", Type: "database"},
	}
	card := renderSummaryCard(ctx, entries)
	for _, want := range []string{"my-app", "React", "Gin", "PostgreSQL", "Drizzle", "Monorepo", "Docker"} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q\ncard:\n%s", want, card)
		}
	}
}

func TestRenderSummaryCardWordPress(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName: "my-wp",
		OutputMode:  "wordpress",
		EnvMode:     "docker",
		FrontendPort: 8080,
		DatabaseIDs: []string{"mysql"},
		DBConfigs:   map[string]registry.DBConfig{"mysql": {Port: 3306}},
	}
	card := renderSummaryCard(ctx, nil)
	if !strings.Contains(card, "my-wp") {
		t.Errorf("card missing project name\ncard:\n%s", card)
	}
	if strings.Contains(card, "Frontend") || strings.Contains(card, "Backend") {
		t.Errorf("wordpress card should not have Frontend/Backend rows\ncard:\n%s", card)
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./cmd/valla/... -run TestRenderSummaryCard -v`
Expected: compile error — `renderSummaryCard` undefined

- [ ] **Step 3: Add `renderSummaryCard` to `main.go`**

Add this function to `cmd/valla/main.go` (before `func main`):

```go
// renderSummaryCard returns a lipgloss-bordered box summarising the user's choices.
// It is displayed in main.go after the TUI exits, before scaffolding begins.
func renderSummaryCard(ctx registry.WeldContext, entries []registry.Entry) string {
	header := itui.StyleCardHeader.Render("valla  ·  " + ctx.ProjectName)

	var rows []string

	if ctx.OutputMode == "wordpress" {
		mysqlCfg := ctx.DBConfigs["mysql"]
		rows = append(rows,
			itui.StyleCardLabel.Render("Mode")+" "+itui.StyleCardValue.Render("WordPress · Docker"),
			itui.StyleCardLabel.Render("WordPress")+" "+itui.StyleCardValue.Render(fmt.Sprintf("port %d", ctx.FrontendPort)),
			itui.StyleCardLabel.Render("MySQL")+" "+itui.StyleCardValue.Render(fmt.Sprintf("port %d", mysqlCfg.Port)),
		)
	} else {
		modeVal := capitalize(ctx.OutputMode) + " · " + capitalize(ctx.EnvMode)
		rows = append(rows, itui.StyleCardLabel.Render("Mode")+" "+itui.StyleCardValue.Render(modeVal))

		if ctx.FrontendID != "" {
			feEntry, _ := registry.FindByID(entries, ctx.FrontendID)
			rows = append(rows, itui.StyleCardLabel.Render("Frontend")+" "+itui.StyleCardValue.Render(feEntry.Name+" ("+feEntry.Runtime+")"))
		}
		if ctx.BackendID != "" {
			beEntry, _ := registry.FindByID(entries, ctx.BackendID)
			rows = append(rows, itui.StyleCardLabel.Render("Backend")+" "+itui.StyleCardValue.Render(beEntry.Name+" ("+beEntry.Runtime+")"))
		}
		if len(ctx.DatabaseIDs) > 0 {
			var dbNames []string
			for _, id := range ctx.DatabaseIDs {
				e, _ := registry.FindByID(entries, id)
				dbNames = append(dbNames, e.Name)
			}
			rows = append(rows, itui.StyleCardLabel.Render("Database")+" "+itui.StyleCardValue.Render(strings.Join(dbNames, " + ")))
		}
		ormLabel := "None"
		if ctx.ORMID == "prisma" {
			ormLabel = "Prisma"
		} else if ctx.ORMID == "drizzle" {
			ormLabel = "Drizzle"
		}
		if len(ctx.DatabaseIDs) > 0 {
			rows = append(rows, itui.StyleCardLabel.Render("ORM")+" "+itui.StyleCardValue.Render(ormLabel))
		}
	}

	body := header + "\n\n" + strings.Join(rows, "\n")
	return itui.StyleCardBorder.Render(body)
}

// capitalize title-cases the first letter of s.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
```

- [ ] **Step 4: Call `renderSummaryCard` in `main()` after `ctx := tuiModel.Context()`**

After the line `ctx := tuiModel.Context()` in `main()`, add:
```go
fmt.Println()
fmt.Println(renderSummaryCard(ctx, entries))
fmt.Println()
```

- [ ] **Step 5: Run the test to confirm it passes**

Run: `go test ./cmd/valla/... -run TestRenderSummaryCard -v`
Expected: PASS

- [ ] **Step 6: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/valla/main.go cmd/valla/main_test.go
git commit -m "feat: add stack summary card rendered before scaffolding"
```

---

### Task 6: Named stdout spinner stages and subprocess stdout suppression

**Files:**
- Modify: `cmd/valla/main.go`
- Modify: `cmd/valla/main.go` — `runScaffold()` function signature

The spinner is a simple `\r` overwrite on stdout. Subprocess output is suppressed during scaffolding (redirected to a buffer, surfaced only on error).

- [ ] **Step 1: Add `printStage` helper to `main.go`**

```go
// printStage prints a named scaffolding stage to stdout using a leading spinner char.
// Call it before each major step; it overwrites the previous line via \r.
func printStage(msg string) {
	fmt.Printf("\r\033[K⠸ %s", msg)
}

// clearStage clears the current spinner line.
func clearStage() {
	fmt.Print("\r\033[K")
}
```

- [ ] **Step 2: Update `runScaffold` to suppress subprocess output**

Change the `runScaffold` signature to accept a `quiet bool` parameter:

```go
func runScaffold(ctx registry.WeldContext, entry registry.Entry, root, targetDir string, quiet bool) error {
```

Inside the function, replace:
```go
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

With:
```go
var stderrBuf strings.Builder
if quiet {
    cmd.Stdout = nil
    cmd.Stderr = &stderrBuf
} else {
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
}
```

And replace:
```go
if err := cmd.Run(); err != nil {
    return fmt.Errorf("scaffold_cmd failed: %w", err)
}
```

With:
```go
if err := cmd.Run(); err != nil {
    if quiet && stderrBuf.Len() > 0 {
        return fmt.Errorf("scaffold_cmd failed: %w\n%s", err, stderrBuf.String())
    }
    return fmt.Errorf("scaffold_cmd failed: %w", err)
}
```

- [ ] **Step 3: Replace plain `fmt.Printf` scaffolding messages in `main()` with spinner stages**

Replace the current scaffolding progress messages in `main()`. For the full-stack path, find:

```go
fmt.Printf("Scaffolding frontend (%s)...\n", frontendEntry.Name)
if err := runScaffold(ctx, frontendEntry, projectRoot, frontendDir); err != nil {
```

Replace with:
```go
printStage(fmt.Sprintf("Scaffolding frontend (%s)...", frontendEntry.Name))
if err := runScaffold(ctx, frontendEntry, projectRoot, frontendDir, true); err != nil {
```

Similarly for backend:
```go
printStage(fmt.Sprintf("Scaffolding backend (%s)...", backendEntry.Name))
if err := runScaffold(ctx, backendEntry, projectRoot, backendDir, true); err != nil {
```

Replace `fmt.Println("Writing .env...")` with:
```go
printStage("Wiring environment...")
```

Replace `fmt.Println("Writing docker-compose.yml...")` (inside the `if ctx.EnvMode == "docker"` block) with:
```go
printStage("Generating Docker config...")
```

Replace individual wiring print statements as follows:

- Remove `fmt.Println("Configuring CORS...")` — CORS patching runs silently (it's fast, no stage needed)
- Remove `fmt.Println("Configuring HTTP client...")` — same, runs silently
- Replace `fmt.Println("Writing prisma/schema.prisma...")` and `fmt.Println("Writing drizzle.config.ts and src/db/index.ts...")` with a single stage gate before the ORM block:

```go
if ctx.ORMID != "" {
    printStage("Injecting ORM config...")
}
```

After all scaffolding is complete (just before the success output block), add:
```go
clearStage()
```

- [ ] **Step 4: Build to verify**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 5: Run tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: add named stdout spinner stages and suppress scaffold subprocess output"
```

---

### Task 7: Styled success output

**Files:**
- Modify: `cmd/valla/main.go`

Replace the large `fmt.Printf` success block at the end of `main()` with styled output using the `Style*` vars exported from `internal/tui/styles.go`.

- [ ] **Step 1: Write a test for `renderSuccessOutput`**

Add to `cmd/valla/main_test.go`:

```go
func TestRenderSuccessOutputMonorepoDocker(t *testing.T) {
	ctx := registry.WeldContext{
		ProjectName: "my-app",
		FrontendID:  "react-node",
		BackendID:   "gin",
		OutputMode:  "monorepo",
		EnvMode:     "docker",
	}
	feEntry := registry.Entry{ID: "react-node", Name: "React", Runtime: "node"}
	beEntry := registry.Entry{ID: "gin", Name: "Gin", Runtime: "go"}
	out := renderSuccessOutput(ctx, "frontend", "backend", feEntry, beEntry, "")
	for _, want := range []string{"Done", "my-app", "docker"} {
		if !strings.Contains(out, want) {
			t.Errorf("success output missing %q\noutput:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./cmd/valla/... -run TestRenderSuccessOutput -v`
Expected: compile error — `renderSuccessOutput` undefined

- [ ] **Step 3: Add `renderSuccessOutput` to `main.go`**

```go
// renderSuccessOutput returns the styled completion message printed after scaffolding.
// ormInstructions is the raw string from ormInstallInstructions (may be empty).
func renderSuccessOutput(ctx registry.WeldContext, frontendDir, backendDir string, feEntry, beEntry registry.Entry, ormInstructions string) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(itui.StyleSuccessHeader.Render("✓  Done!"))
	sb.WriteString("\n\n")

	// Directory tree
	if ctx.OutputMode == "wordpress" {
		tree := fmt.Sprintf("%s/\n  wordpress/\n  .env\n  docker-compose.yml", ctx.ProjectName)
		sb.WriteString(itui.StyleSuccessTree.Render(tree))
	} else if ctx.OutputMode == "separate" {
		tree := fmt.Sprintf("%s/\n%s/\n.env", frontendDir, backendDir)
		if ctx.EnvMode == "docker" {
			tree += "\ndocker-compose.yml"
		}
		sb.WriteString(itui.StyleSuccessTree.Render(tree))
	} else {
		var children []string
		if ctx.FrontendID != "" {
			children = append(children, "  "+frontendDir+"/")
		}
		if ctx.BackendID != "" {
			children = append(children, "  "+backendDir+"/")
		}
		children = append(children, "  .env")
		if ctx.EnvMode == "docker" {
			children = append(children, "  docker-compose.yml")
		}
		tree := ctx.ProjectName + "/\n" + strings.Join(children, "\n")
		sb.WriteString(itui.StyleSuccessTree.Render(tree))
	}

	sb.WriteString("\n\n")

	// Next steps
	sb.WriteString("Next steps:\n\n")
	if ctx.OutputMode == "wordpress" {
		sb.WriteString(itui.StyleSuccessCmd.Render(fmt.Sprintf("  cd %s && docker-compose up -d", ctx.ProjectName)))
		sb.WriteString(fmt.Sprintf("\n\nThen open http://localhost:%d to finish WordPress setup.", ctx.FrontendPort))
	} else if ctx.EnvMode == "docker" {
		if ctx.OutputMode == "separate" {
			sb.WriteString(itui.StyleSuccessCmd.Render("  docker-compose up -d"))
		} else {
			sb.WriteString(itui.StyleSuccessCmd.Render(fmt.Sprintf("  cd %s && docker-compose up -d", ctx.ProjectName)))
		}
	} else {
		if ctx.OutputMode != "separate" {
			sb.WriteString(itui.StyleSuccessCmd.Render(fmt.Sprintf("  cd %s", ctx.ProjectName)))
			sb.WriteString("\n")
		}
		if ctx.FrontendID != "" {
			sb.WriteString(itui.StyleSuccessCmd.Render(fmt.Sprintf("  cd %s && npm install", frontendDir)))
			sb.WriteString("\n")
		}
		if ctx.BackendID != "" {
			cmd := localRunCmd(beEntry, backendDir)
			sb.WriteString(itui.StyleSuccessCmd.Render("  " + cmd))
			sb.WriteString("\n")
		}
	}

	if ormInstructions != "" {
		sb.WriteString("\n")
		sb.WriteString(ormInstructions)
	}

	sb.WriteString("\n")
	return sb.String()
}

// localRunCmd returns the run command string for a backend in local (.env) mode.
func localRunCmd(entry registry.Entry, dir string) string {
	switch entry.Runtime {
	case "go":
		return fmt.Sprintf("cd %s && go run main.go", dir)
	case "python3":
		return fmt.Sprintf("cd %s && source venv/bin/activate && python ...", dir)
	case "dotnet":
		return fmt.Sprintf("cd %s && dotnet run", dir)
	case "java":
		switch entry.ID {
		case "java-springboot-maven":
			return fmt.Sprintf("cd %s && mvn spring-boot:run", dir)
		case "java-springboot-gradle":
			return fmt.Sprintf("cd %s && ./gradlew bootRun", dir)
		case "java-quarkus-maven":
			return fmt.Sprintf("cd %s && mvn quarkus:dev", dir)
		case "java-quarkus-gradle":
			return fmt.Sprintf("cd %s && ./gradlew quarkusDev", dir)
		}
	}
	return fmt.Sprintf("cd %s && npm install && npm start", dir)
}
```

- [ ] **Step 4: Replace the old success output block in `main()`**

Delete the large `fmt.Printf` success/next-steps block (lines ~254–323 in the original file) and replace with a single call:

```go
var ormInstr string
if ctx.ORMID != "" {
    ormInstr = ormInstallInstructions(ctx.ORMID, primarySQLDB(ctx.DatabaseIDs))
}
fmt.Print(renderSuccessOutput(ctx, frontendDir, backendDir, frontendEntry, backendEntry, ormInstr))
```

Also replace the WordPress success block (`fmt.Printf("\nWordPress project scaffolded successfully...")`) with:
```go
fmt.Print(renderSuccessOutput(ctx, "", "", registry.Entry{}, registry.Entry{}, ""))
return
```

> **Important:** The WordPress path in `main()` uses `return` after its success output to prevent falling through into the monorepo/separate scaffolding path. Keep the `return` after `fmt.Print(renderSuccessOutput(...))` — do not remove it.

- [ ] **Step 5: Run the test to confirm it passes**

Run: `go test ./cmd/valla/... -run TestRenderSuccessOutput -v`
Expected: PASS

- [ ] **Step 6: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/valla/main.go cmd/valla/main_test.go
git commit -m "feat: replace plain success output with styled completion message"
```

---

### Task 8: Update checker goroutine

**Files:**
- Modify: `cmd/valla/main.go`

The goroutine is launched after `tea.NewProgram` assigns `program`, captures `program` by closure, and calls `program.Send(itui.UpdateAvailableMsg{...})`.

- [ ] **Step 1: Write a test for `checkLatestVersion`**

Add to `cmd/valla/main_test.go`:

```go
func TestParseLatestVersion(t *testing.T) {
	// parseTagFromBody parses {"tag_name":"v1.2.3"} → "v1.2.3"
	body := `{"tag_name":"v1.2.3","name":"Release v1.2.3"}`
	got := parseTagFromBody([]byte(body))
	if got != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", got)
	}
}

func TestParseLatestVersionMissing(t *testing.T) {
	got := parseTagFromBody([]byte(`{}`))
	if got != "" {
		t.Errorf("expected empty string for missing tag_name, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

Run: `go test ./cmd/valla/... -run TestParseLatestVersion -v`
Expected: compile error — `parseTagFromBody` undefined

- [ ] **Step 3: Add update checker helpers and goroutine launch to `main.go`**

Add imports `"encoding/json"`, `"net/http"`, `"time"` (if not already present).

Add helper functions:

```go
// parseTagFromBody extracts the tag_name field from a GitHub releases API JSON response.
func parseTagFromBody(body []byte) string {
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.TagName
}

// startUpdateChecker launches a goroutine that checks GitHub for a newer release
// and sends UpdateAvailableMsg to program if one is found.
// It is a no-op if VALLA_NO_UPDATE_CHECK=1 is set or if version == "dev".
func startUpdateChecker(program *tea.Program, currentVersion string) {
	if os.Getenv("VALLA_NO_UPDATE_CHECK") == "1" || currentVersion == "dev" {
		return
	}
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/tariktz/valla/releases/latest")
		if err != nil {
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		latest := parseTagFromBody(body)
		if latest != "" && latest != currentVersion {
			program.Send(itui.UpdateAvailableMsg{Version: latest})
		}
	}()
}
```

In `main()`, after `program := tea.NewProgram(model)` and before `program.Run()`, add:

```go
startUpdateChecker(program, version)
```

- [ ] **Step 4: Run test to confirm it passes**

Run: `go test ./cmd/valla/... -run TestParseLatestVersion -v`
Expected: PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/valla/main.go cmd/valla/main_test.go
git commit -m "feat: add update checker goroutine with program.Send on new release"
```

---

### Task 9: `--help` cleanup

**Files:**
- Modify: `cmd/valla/main.go`

- [ ] **Step 1: Add custom `flag.Usage` in `main()`**

Add `"flag"` to imports.

Replace the `--version` check added in Task 2 with an expanded manual args handler that covers both `--version` and `--help`. This avoids using `flag.Parse()` entirely (which would intercept unregistered flags). At the very start of `main()`, add:

```go
usage := func() {
    fmt.Fprintf(os.Stderr, `valla-cli — Scaffold your full stack in seconds.

Usage:
  valla-cli           Run the interactive wizard
  valla-cli --version Print version and exit
  valla-cli --help    Show this help

Environment:
  VALLA_NO_UPDATE_CHECK=1  Disable the update checker

`)
}

if len(os.Args) > 1 {
    switch os.Args[1] {
    case "--version", "-version":
        fmt.Printf("valla-cli %s\n", version)
        return
    case "--help", "-help", "-h":
        usage()
        return
    }
}
```

Remove the separate `--version` check added in Task 2 — it is superseded by this block.

- [ ] **Step 2: Verify `--help` output**

Run: `go run ./cmd/valla --help`
Expected: custom help text with the valla tagline, flags, and env var note

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: add custom --help usage message"
```

---

## Chunk 4: Demo GIF and README

### Task 10: VHS tape script

**Files:**
- Create: `vhs/demo.tape`

> **Prerequisite:** VHS must be installed (`brew install vhs` or `go install github.com/charmbracelet/vhs@latest`). Run `valla-cli` against the polished build to confirm the TUI looks correct before recording.

- [ ] **Step 1: Create the `vhs/` directory and tape file**

```
# vhs/demo.tape
Output docs/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 800
Set Height 500
Set Framerate 24
Set PlaybackSpeed 1.0

# Start valla-cli
Type "valla-cli"
Enter
Sleep 1500ms

# Project name
Type "demo-app"
Enter
Sleep 500ms

# Output structure: Monorepo (first option, just Enter)
Enter
Sleep 500ms

# Frontend runtime: Node (first available, Enter)
Enter
Sleep 500ms

# Frontend framework: React (first option, Enter)
Enter
Sleep 500ms

# Backend runtime: Go (first option, Enter)
Enter
Sleep 500ms

# Backend framework: Gin (first option, Enter)
Enter
Sleep 500ms

# Database: PostgreSQL — press Space to select, then Enter
Space
Enter
Sleep 500ms

# ORM: None (first option, Enter)
Enter
Sleep 500ms

# Env mode: Docker (second option — Down then Enter)
Down
Enter
Sleep 500ms

# Port overrides: accept frontend port default, then backend port default
Enter
Sleep 300ms
Enter
Sleep 500ms

# Confirm
Enter
Sleep 3000ms
```

- [ ] **Step 2: Record the GIF (manual step)**

```bash
mkdir -p docs
vhs vhs/demo.tape
```

Expected: `docs/demo.gif` created, ~25–35 seconds runtime

- [ ] **Step 3: Verify the GIF looks correct**

Open `docs/demo.gif` in a browser or image viewer and confirm: banner shows, TUI flow is clear, summary card and success output appear.

- [ ] **Step 4: Commit**

```bash
git add vhs/demo.tape docs/demo.gif
git commit -m "docs: add VHS tape script and demo GIF"
```

---

### Task 11: README overhaul

**Files:**
- Modify: `README.md`

> **Prerequisite:** CI workflow must have run successfully on `main` at least once before adding the CI badge (otherwise badge shows 404).

- [ ] **Step 1: Rewrite the top of `README.md`**

Replace everything above the existing "Supported stacks" section with:

```markdown
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

![valla demo](docs/demo.gif)

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
```

- [ ] **Step 2: Move architecture and contributing sections to the bottom**

Ensure the README structure is:
1. Banner + badges + GIF
2. Intro paragraph
3. Quickstart
4. Supported stacks table (keep existing)
5. What gets generated (directory tree examples — keep existing)
6. Requirements (keep existing)
7. Roadmap (keep existing)
8. Architecture (move to bottom)
9. Contributing (move to bottom)

- [ ] **Step 3: Verify README renders correctly**

Open the file and visually scan: banner is at top, GIF is visible, quickstart is prominent.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: overhaul README with banner, demo GIF, badges, and improved structure"
```

---

## Verification

After all tasks are complete:

- [ ] `go test ./...` passes
- [ ] `go build ./...` produces a binary
- [ ] `./valla-cli --version` prints `valla-cli dev` (locally)
- [ ] `./valla-cli --help` shows custom usage
- [ ] Running `./valla-cli` shows the new multi-line ASCII art banner
- [ ] After selecting a stack and confirming, the summary card appears
- [ ] Spinner stages appear during scaffolding
- [ ] Styled success output (green ✓ Done!) appears at the end
- [ ] CI workflow is green on GitHub
- [ ] README looks correct on GitHub with GIF loading
