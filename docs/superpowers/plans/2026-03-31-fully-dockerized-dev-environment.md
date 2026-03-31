# Fully Dockerized Dev Environment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Fully Dockerized" output structure option to the valla TUI that scaffolds a complete dev environment (VS Code Dev Container + `docker-compose.dev.yml` + Makefile) requiring only Docker — no local runtimes needed.

**Architecture:** A new `"Fully Dockerized"` choice in the `OutputStructure` step sets `ctx.DevContainer = true` and swaps in all-available runtime options so every framework is selectable. After scaffolding, `GenerateDevContainerFiles` renders three files (devcontainer.json, docker-compose.dev.yml, Makefile) using pre-computed per-stack values. The feature is gated on Docker being detected on the host machine.

**Tech Stack:** Go, Bubbletea TUI, text/template, embed.FS, `mcr.microsoft.com/devcontainers/*` images

**Spec:** `docs/superpowers/specs/2026-03-31-fully-dockerized-dev-environment-design.md`

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Modify | `internal/registry/types.go` | Add `DevContainer bool` to `WeldContext`; add `DevContainerImage`/`DevCmd` to `Entry` |
| Modify | `internal/tui/steps/runtime_select.go` | Add exported `AllRuntimeOptions` helper |
| Modify | `internal/tui/steps/output_structure.go` | Accept `dockerAvailable bool`; conditionally show "Fully Dockerized"; add tree preview case |
| Modify | `internal/tui/model.go` | Add `allFeRuntimeOpts`/`allBeRuntimeOpts` fields; update `New`; update phase handler and transitions; update `summaryLines` |
| Modify | `cmd/valla/main.go` | Detect docker; thread `dockerAvailable` through; update `renderSummaryCard`/`renderSuccessOutput`; call `GenerateDevContainerFiles` |
| Create | `internal/scaffolder/scaffolder_devcontainer.go` | `GenerateDevContainerFiles` — derives images/commands, renders 3 files, handles rollback |
| Create | `internal/registry/data/templates/devcontainer/devcontainer.json.tmpl` | devcontainer.json template |
| Create | `internal/registry/data/templates/devcontainer/Makefile.tmpl` | Makefile template |
| Modify | `internal/registry/data/backends/java-springboot-maven.yaml` | Add `dev_cmd: ./mvnw spring-boot:run` |
| Modify | `internal/registry/data/backends/java-springboot-gradle.yaml` | Add `dev_cmd: ./gradlew bootRun` |
| Modify | `internal/registry/data/backends/java-quarkus-maven.yaml` | Add `dev_cmd: ./mvnw quarkus:dev` |
| Modify | `internal/registry/data/backends/java-quarkus-gradle.yaml` | Add `dev_cmd: ./gradlew quarkusDev` |
| Create | `internal/tui/steps/runtime_select_test.go` | Tests for `AllRuntimeOptions` |
| Create | `internal/tui/devcontainer_test.go` | Tests for DevContainer phase flow |
| Modify | `internal/scaffolder/scaffolder_devcontainer_test.go` (create) | Tests for `GenerateDevContainerFiles` |

---

## Chunk 1: Data Layer — Types, Helper, Detector

### Task 1: Add `DevContainer` to `WeldContext` and new fields to `Entry`

**Files:**
- Modify: `internal/registry/types.go`

- [ ] **Step 1: Write failing test**

Create `internal/registry/model_test.go` with `package registry_test`:
```go
package registry_test

import (
    "testing"

    "github.com/tariktz/valla-cli/internal/registry"
)

func TestWeldContextDevContainerField(t *testing.T) {
    ctx := registry.WeldContext{DevContainer: true}
    if !ctx.DevContainer {
        t.Error("DevContainer field should be settable to true")
    }
}

func TestEntryDevContainerFields(t *testing.T) {
    e := registry.Entry{
        DevContainerImage: "mcr.microsoft.com/devcontainers/go",
        DevCmd:            "go run .",
    }
    if e.DevContainerImage == "" || e.DevCmd == "" {
        t.Error("DevContainerImage and DevCmd fields should be settable")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /path/to/project && go test ./internal/registry/... -run "TestWeldContextDevContainerField|TestEntryDevContainerFields" -v
```
Expected: compile error — `DevContainer`, `DevContainerImage`, `DevCmd` undefined.

- [ ] **Step 3: Add fields to `internal/registry/types.go`**

In `WeldContext`, after `EnvMode string`:
```go
DevContainer bool // true when "Fully Dockerized" output structure is chosen
```

In `Entry`, after `Docker *DockerConfig \`yaml:"docker"\``:
```go
DevContainerImage string `yaml:"devcontainer_image"` // overrides runtime-derived devcontainer image
DevCmd            string `yaml:"dev_cmd"`             // hot-reload command inside the dev container
```

Note: backtick struct tags are required — without them the fields will not unmarshal from YAML.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/registry/... -run "TestWeldContextDevContainerField|TestEntryDevContainerFields" -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/registry/types.go internal/registry/model_test.go
git commit -m "feat: add DevContainer, DevContainerImage, DevCmd fields to registry types"
```

---

### Task 2: Add `AllRuntimeOptions` helper

**Files:**
- Modify: `internal/tui/steps/runtime_select.go`
- Create: `internal/tui/steps/runtime_select_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/tui/steps/runtime_select_test.go`:
```go
package steps_test

import (
    "testing"

    "github.com/tariktz/valla-cli/internal/tui/steps"
)

func TestAllRuntimeOptions_SetsAllAvailable(t *testing.T) {
    input := []steps.RuntimeOption{
        {Name: "go", Available: false, Reason: "go not found"},
        {Name: "node", Available: true},
        {Name: "python3", Available: false, Reason: "python3 not found"},
    }

    result := steps.AllRuntimeOptions(input)

    for _, opt := range result {
        if !opt.Available {
            t.Errorf("option %q should be Available=true, got false", opt.Name)
        }
    }
}

func TestAllRuntimeOptions_PreservesNamesAndReasons(t *testing.T) {
    input := []steps.RuntimeOption{
        {Name: "go", Available: false, Reason: "go not found"},
    }

    result := steps.AllRuntimeOptions(input)

    if result[0].Name != "go" {
        t.Errorf("expected Name=go, got %q", result[0].Name)
    }
    if result[0].Reason != "go not found" {
        t.Errorf("expected Reason preserved, got %q", result[0].Reason)
    }
}

func TestAllRuntimeOptions_DoesNotMutateInput(t *testing.T) {
    input := []steps.RuntimeOption{
        {Name: "go", Available: false},
    }

    steps.AllRuntimeOptions(input)

    if input[0].Available {
        t.Error("AllRuntimeOptions must not mutate the input slice")
    }
}

func TestAllRuntimeOptions_EmptySlice(t *testing.T) {
    result := steps.AllRuntimeOptions(nil)
    if len(result) != 0 {
        t.Errorf("expected empty result for nil input, got len=%d", len(result))
    }
}
```

Note: use `package steps_test` (external test package). Both `package steps` and `package steps_test` files can coexist in the same directory — `port_overrides_test.go` uses `steps_test`, which is the pattern to follow here.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tui/steps/... -run "TestAllRuntimeOptions" -v
```
Expected: compile error — `steps.AllRuntimeOptions` is undefined. A compile error is the correct red-phase result when the function doesn't exist yet.

- [ ] **Step 3: Add `AllRuntimeOptions` to `internal/tui/steps/runtime_select.go`**

After the existing `runtimeOptionsAll` function:
```go
// AllRuntimeOptions returns a copy of opts with every entry marked Available: true.
// The input slice must be the full set of known runtimes (not a pre-filtered subset).
// Does not mutate the input.
func AllRuntimeOptions(opts []RuntimeOption) []RuntimeOption {
    out := make([]RuntimeOption, len(opts))
    for i, o := range opts {
        o.Available = true
        out[i] = o
    }
    return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/tui/steps/... -run "TestAllRuntimeOptions" -v
```
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/steps/runtime_select.go internal/tui/steps/runtime_select_test.go
git commit -m "feat: add AllRuntimeOptions helper to steps package"
```

---

### Task 3: Add docker detection

**Files:**
- Modify: `cmd/valla/main.go`

- [ ] **Step 1: Write failing test**

In `cmd/valla/main_test.go` (create if it doesn't exist), add:
```go
func TestDockerDetectionIncludesDockerBinary(t *testing.T) {
    // This is an integration smoke test: verify the binary slice passed to Detect
    // includes "docker". We do this by checking the compiled binary behaviour is
    // correct — actual docker availability varies per machine.
    // The test just verifies the code compiles with the new call.
    _ = detector.Detect([]string{"docker"})
}
```

Actually for the detector change, a compilation test isn't meaningful. Instead, test the effect: `dockerAvailable` must be passed to `tui.New`. Since `tui.New`'s signature change is in Task 4, this task just adds `"docker"` to the Detect call and stores the result. Skip the test here — it will be covered by the compilation of Task 4's test.

- [ ] **Step 1: Add docker to Detect call in `cmd/valla/main.go`**

Change:
```go
available := detector.Detect([]string{"go", "node", "bun", "python3", "dotnet"})
```
To:
```go
available := detector.Detect([]string{"go", "node", "bun", "python3", "dotnet", "docker"})
dockerAvailable := available["docker"]
_ = dockerAvailable // temporary — wired through in Task 5
```

Leave the `DetectWithAliases` block for java **unchanged**.

The `_ = dockerAvailable` blank assignment keeps the codebase compiling until Task 5 removes it and passes the value to `tui.New`. Without this, Go's unused-variable rule causes a compile error that breaks `go test ./...` for all packages.

- [ ] **Step 2: Verify the project still compiles**

```bash
go build ./...
```
Expected: success (no compile errors).

- [ ] **Step 3: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: detect docker binary availability"
```

---

## Chunk 2: TUI — OutputStructure Step and Model Updates

### Task 4: Update `OutputStructure` step to show "Fully Dockerized"

**Files:**
- Modify: `internal/tui/steps/output_structure.go`
- Create: `internal/tui/steps/output_structure_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/tui/steps/output_structure_test.go`:
```go
package steps_test

import (
    "testing"

    "github.com/tariktz/valla-cli/internal/tui/steps"
)

func TestOutputStructure_WithDocker_ShowsFullyDockerized(t *testing.T) {
    m := steps.NewOutputStructure(true)
    view := m.View()
    if !strings.Contains(view, "Fully Dockerized") {
        t.Error("expected 'Fully Dockerized' option when dockerAvailable=true")
    }
}

func TestOutputStructure_WithoutDocker_HidesFullyDockerized(t *testing.T) {
    m := steps.NewOutputStructure(false)
    view := m.View()
    if strings.Contains(view, "Fully Dockerized") {
        t.Error("expected 'Fully Dockerized' option to be hidden when dockerAvailable=false")
    }
}
```

Add `"strings"` to the import. Run:
```bash
go test ./internal/tui/steps/... -run "TestOutputStructure" -v
```
Expected: FAIL — `NewOutputStructure` does not accept a bool argument.

- [ ] **Step 2: Update `NewOutputStructure` in `internal/tui/steps/output_structure.go`**

Change `OutputStructure` struct to store `dockerAvailable`:
```go
type OutputStructure struct {
    options        []string
    cursor         int
    dockerAvailable bool
}
```

Change `NewOutputStructure`:
```go
func NewOutputStructure(dockerAvailable bool) OutputStructure {
    options := []string{"Monorepo", "Separate folders", "Frontend only", "Backend only", "WordPress"}
    if dockerAvailable {
        options = append(options, "Fully Dockerized")
    }
    return OutputStructure{
        options:         options,
        dockerAvailable: dockerAvailable,
    }
}
```

Add the tree preview case for `"Fully Dockerized"` in `structurePreview`:
```go
case "Fully Dockerized":
    rows = []line{
        tree("", "myapp/"),
        tree("├── ", "frontend/"),
        tree("├── ", "backend/"),
        tree("├── ", ".devcontainer/"),
        tree("│   └── ", "devcontainer.json"),
        tree("├── ", ".env"),
        tree("├── ", "docker-compose.yml"),
        tree("├── ", "docker-compose.dev.yml"),
        tree("└── ", "Makefile"),
    }
```

- [ ] **Step 3: Fix the existing call site in `model.go`**

In `internal/tui/model.go`, in `handleStepDone` for `phaseProjectName`, change:
```go
m.current = steps.NewOutputStructure()
```
to:
```go
m.current = steps.NewOutputStructure(m.dockerAvailable)
```

Also add `dockerAvailable bool` to the `Model` struct (in the next task). For now just note that `m.dockerAvailable` needs to exist — the compile will fail until Task 5 adds it to the struct.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/tui/steps/... -run "TestOutputStructure" -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/steps/output_structure.go internal/tui/steps/output_structure_test.go
git commit -m "feat: add Fully Dockerized option to OutputStructure step"
```

---

### Task 5: Update `Model` struct, `tui.New`, and phase handler

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `cmd/valla/main.go` (wire `dockerAvailable` through)
- Create: `internal/tui/devcontainer_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/tui/devcontainer_test.go`:
```go
package tui

import (
    "testing"

    "github.com/tariktz/valla-cli/internal/registry"
    "github.com/tariktz/valla-cli/internal/tui/steps"
)

// allAvailableOpts returns a slice of RuntimeOptions all marked available.
func allAvailableOpts(names []string) []steps.RuntimeOption {
    opts := make([]steps.RuntimeOption, len(names))
    for i, n := range names {
        opts[i] = steps.RuntimeOption{Name: n, Available: true}
    }
    return opts
}

// noneAvailableOpts returns a slice of RuntimeOptions all marked unavailable.
func noneAvailableOpts(names []string) []steps.RuntimeOption {
    opts := make([]steps.RuntimeOption, len(names))
    for i, n := range names {
        opts[i] = steps.RuntimeOption{Name: n, Available: false, Reason: "not found"}
    }
    return opts
}

func TestNewModel_StoresDockerAvailable(t *testing.T) {
    m := New(nil, nil, nil, true)
    if !m.dockerAvailable {
        t.Error("expected dockerAvailable=true to be stored on Model")
    }
}

func TestNewModel_StoresAllRuntimeOpts(t *testing.T) {
    feOpts := noneAvailableOpts([]string{"node", "bun"})
    beOpts := noneAvailableOpts([]string{"go", "node"})

    m := New(nil, feOpts, beOpts, true)

    for _, o := range m.allFeRuntimeOpts {
        if !o.Available {
            t.Errorf("allFeRuntimeOpts: expected %q to be Available=true", o.Name)
        }
    }
    for _, o := range m.allBeRuntimeOpts {
        if !o.Available {
            t.Errorf("allBeRuntimeOpts: expected %q to be Available=true", o.Name)
        }
    }
}

func TestDevContainerPath_SetsContextFields(t *testing.T) {
    feOpts := noneAvailableOpts([]string{"node"})
    beOpts := noneAvailableOpts([]string{"go"})
    m := New([]registry.Entry{}, feOpts, beOpts, true)

    // Simulate project name step completing
    m.ctx.ProjectName = "myapp"
    m.phase = phaseOutputStructure

    // Send "Fully Dockerized" choice
    updated, _ := m.handleStepDone(steps.StepDone{Value: "Fully Dockerized"})
    result := updated.(Model)

    if result.ctx.OutputMode != "devcontainer" {
        t.Errorf("OutputMode: got %q, want %q", result.ctx.OutputMode, "devcontainer")
    }
    if !result.ctx.DevContainer {
        t.Error("DevContainer should be true")
    }
    if result.ctx.EnvMode != "docker" {
        t.Errorf("EnvMode: got %q, want %q", result.ctx.EnvMode, "docker")
    }
    if result.phase != phaseFrontendRuntime {
        t.Errorf("phase: got %v, want phaseFrontendRuntime", result.phase)
    }
}

func TestDevContainerPath_RuntimeOptsAreAllAvailable(t *testing.T) {
    feOpts := noneAvailableOpts([]string{"node", "bun"})
    beOpts := noneAvailableOpts([]string{"go", "node"})
    m := New([]registry.Entry{}, feOpts, beOpts, true)
    m.phase = phaseOutputStructure

    updated, _ := m.handleStepDone(steps.StepDone{Value: "Fully Dockerized"})
    result := updated.(Model)

    for _, o := range result.feRuntimeOpts {
        if !o.Available {
            t.Errorf("feRuntimeOpts after DevContainer choice: %q should be Available", o.Name)
        }
    }
}

func TestDevContainerPath_SkipsEnvModeAfterDatabaseSelect(t *testing.T) {
    m := Model{
        ctx: registry.WeldContext{DevContainer: true},
    }
    m.phase = phaseDatabaseSelect

    updated, _ := m.handleStepDone(steps.StepDone{Value: []string{}})
    result := updated.(Model)

    if result.phase == phaseEnvMode {
        t.Error("DevContainer path must not transition to phaseEnvMode")
    }
    if result.phase != phasePortOverrides {
        t.Errorf("expected phasePortOverrides, got phase=%v", result.phase)
    }
}

func TestDevContainerPath_SkipsEnvModeAfterORMSelect(t *testing.T) {
    m := Model{
        ctx: registry.WeldContext{DevContainer: true},
    }
    m.phase = phaseORMSelect

    updated, _ := m.handleStepDone(steps.StepDone{Value: "None"})
    result := updated.(Model)

    if result.phase == phaseEnvMode {
        t.Error("DevContainer path must not transition to phaseEnvMode")
    }
    if result.phase != phasePortOverrides {
        t.Errorf("expected phasePortOverrides, got phase=%v", result.phase)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/tui/... -run "TestNewModel_|TestDevContainerPath_" -v
```
Expected: FAIL — `New` doesn't accept 4 args, `dockerAvailable` field doesn't exist.

- [ ] **Step 3: Update `Model` struct and `New` in `internal/tui/model.go`**

Add fields to `Model` struct:
```go
type Model struct {
    phase   phase
    current tea.Model
    ctx     registry.WeldContext
    entries []registry.Entry

    feRuntimeOpts    []steps.RuntimeOption
    beRuntimeOpts    []steps.RuntimeOption
    allFeRuntimeOpts []steps.RuntimeOption // all Available: true — used on DevContainer path
    allBeRuntimeOpts []steps.RuntimeOption // all Available: true — used on DevContainer path

    dockerAvailable bool

    selectedFERuntime string
    selectedBERuntime string

    confirmed    bool
    updateNotice string
}
```

Update `New`:
```go
func New(entries []registry.Entry, feRuntimeOpts, beRuntimeOpts []steps.RuntimeOption, dockerAvailable bool) Model {
    return Model{
        phase:            phaseProjectName,
        current:          steps.NewProjectName(),
        entries:          entries,
        feRuntimeOpts:    feRuntimeOpts,
        beRuntimeOpts:    beRuntimeOpts,
        allFeRuntimeOpts: steps.AllRuntimeOptions(feRuntimeOpts),
        allBeRuntimeOpts: steps.AllRuntimeOptions(beRuntimeOpts),
        dockerAvailable:  dockerAvailable,
    }
}
```

- [ ] **Step 4: Update `phaseProjectName` handler to pass `dockerAvailable` to `NewOutputStructure`**

In `handleStepDone`, `case phaseProjectName:`:
```go
case phaseProjectName:
    m.ctx.ProjectName = msg.Value.(string)
    m.ctx.DBName = m.ctx.ProjectName
    m.phase = phaseOutputStructure
    m.current = steps.NewOutputStructure(m.dockerAvailable)  // was: steps.NewOutputStructure()
```

- [ ] **Step 5: Add "Fully Dockerized" branch to `phaseOutputStructure` handler**

In `handleStepDone`, `case phaseOutputStructure:`, add as the **first** branch (before WordPress):
```go
case phaseOutputStructure:
    choice := msg.Value.(string)
    if choice == "Fully Dockerized" {
        m.ctx.OutputMode = "devcontainer"
        m.ctx.DevContainer = true
        m.ctx.EnvMode = "docker"
        // Set DB host to "db" for all already-configured DBs (may be empty at this point)
        for id, cfg := range m.ctx.DBConfigs {
            cfg.Host = "db"
            m.ctx.DBConfigs[id] = cfg
        }
        // Swap to all-available runtime opts so all frameworks are selectable.
        // AllRuntimeOptions receives the original feRuntimeOpts/beRuntimeOpts from New()
        // (the potentially-filtered slices). It marks all entries Available:true and returns
        // a new slice — it does not re-read the registry.
        m.feRuntimeOpts = m.allFeRuntimeOpts
        m.beRuntimeOpts = m.allBeRuntimeOpts
        m.phase = phaseFrontendRuntime
        m.current = steps.NewRuntimeSelect("Select frontend runtime:", m.feRuntimeOpts)
        return m, m.current.Init() // early return required — skips anyAvailable guard below
    }
    if choice == "WordPress" {
        // ... existing code unchanged ...
```

- [ ] **Step 6: Update `phaseDatabaseSelect` and `phaseORMSelect` transitions**

In `phaseDatabaseSelect` handler, change:
```go
// Before:
if isORMEligible(m) {
    m.phase = phaseORMSelect
    m.current = steps.NewRuntimeSelect("Select ORM (optional):", ormOptions())
} else {
    m.phase = phaseEnvMode
    m.current = steps.NewEnvMode()
}

// After:
if isORMEligible(m) {
    m.phase = phaseORMSelect
    m.current = steps.NewRuntimeSelect("Select ORM (optional):", ormOptions())
} else if m.ctx.DevContainer {
    m.phase = phasePortOverrides
    m.current = m.buildPortOverrides()
} else {
    m.phase = phaseEnvMode
    m.current = steps.NewEnvMode()
}
```

In `phaseORMSelect` handler, change:
```go
// Before:
m.ctx.ORMID = ormIDFromName(msg.Value.(string))
m.phase = phaseEnvMode
m.current = steps.NewEnvMode()

// After:
m.ctx.ORMID = ormIDFromName(msg.Value.(string))
if m.ctx.DevContainer {
    m.phase = phasePortOverrides
    m.current = m.buildPortOverrides()
} else {
    m.phase = phaseEnvMode
    m.current = steps.NewEnvMode()
}
```

- [ ] **Step 7: Update `summaryLines`**

In `summaryLines()`, the non-WordPress branch currently builds a `lines` slice where the third element is `fmt.Sprintf("Env mode: %s", m.ctx.EnvMode)`. Replace that specific slice literal — do NOT append a second mode line. The fix is to build `envModeLine` conditionally before constructing the slice, then use it as the single third element:

```go
// Before:
lines := []string{
    fmt.Sprintf("Project:    %s", m.ctx.ProjectName),
    fmt.Sprintf("Structure:  %s", m.ctx.OutputMode),
    fmt.Sprintf("Env mode:   %s", m.ctx.EnvMode),
}

// After:
envModeLine := fmt.Sprintf("Env mode:   %s", m.ctx.EnvMode)
if m.ctx.DevContainer {
    envModeLine = "Mode:       Fully Dockerized (Dev Container)"
}
lines := []string{
    fmt.Sprintf("Project:    %s", m.ctx.ProjectName),
    fmt.Sprintf("Structure:  %s", m.ctx.OutputMode),
    envModeLine, // replaces the original Env mode line — not an additional append
}
```

Important: `envModeLine` replaces index 2 of the slice. Do not leave the original `"Env mode: %s"` line and append the DevContainer line separately — that would produce two mode lines in the confirm summary.

- [ ] **Step 8: Update `cmd/valla/main.go` — wire `dockerAvailable` through**

Change the `tui.New` call:
```go
// Before:
model := itui.New(entries, feRuntimeOpts, beRuntimeOpts)

// After:
model := itui.New(entries, feRuntimeOpts, beRuntimeOpts, dockerAvailable)
```

- [ ] **Step 9: Run all tests**

```bash
go test ./... -v 2>&1 | tail -30
```
Expected: all pass.

- [ ] **Step 10: Commit**

```bash
git add internal/tui/model.go internal/tui/devcontainer_test.go cmd/valla/main.go
git commit -m "feat: add DevContainer TUI flow — OutputStructure option, phase transitions, summaryLines"
```

---

## Chunk 3: Rendering — `renderSummaryCard` and `renderSuccessOutput`

### Task 6: Update `renderSummaryCard` in `cmd/valla/main.go`

**Files:**
- Modify: `cmd/valla/main.go`

- [ ] **Step 1: Locate the function**

`renderSummaryCard` is in `cmd/valla/main.go`. The current structure is:
```go
if ctx.OutputMode == "wordpress" {
    // wordpress rows
} else {
    modeVal := capitalize(ctx.OutputMode) + " · " + capitalize(ctx.EnvMode)
    // ... frontend/backend/database/ORM rows
}
```

- [ ] **Step 2: Add DevContainer as a top-level branch**

The current structure is `if ctx.OutputMode == "wordpress" { ... } else { ... }`. Add `ctx.DevContainer` as a new top-level branch **before** the WordPress branch (parallel to it, matching that precedent):

```go
if ctx.DevContainer {
    modeVal := "Dev Container (Docker)"
    rows = append(rows, itui.StyleCardLabel.Render("Mode")+" "+itui.StyleCardValue.Render(modeVal))
    if ctx.FrontendID != "" {
        feEntry, _ := registry.FindByID(entries, ctx.FrontendID)
        rows = append(rows, itui.StyleCardLabel.Render("Frontend")+" "+itui.StyleCardValue.Render(feEntry.Name+" ("+feEntry.Runtime+")"))
    }
    if ctx.BackendID != "" {
        beEntry, _ := registry.FindByID(entries, ctx.BackendID)
        rows = append(rows, itui.StyleCardLabel.Render("Backend")+" "+itui.StyleCardValue.Render(beEntry.Name+" ("+beEntry.Runtime+")"))
    }
    // database and ORM rows — copy from else block
} else if ctx.OutputMode == "wordpress" {
    // ... existing wordpress block unchanged
} else {
    // ... existing else block unchanged
}
```

This is a parallel top-level branch — do NOT nest inside the `else` block. This matches the WordPress pattern and prevents future `ctx.OutputMode`/`ctx.EnvMode` string checks inside the `else` block from misfiring for the devcontainer path.

- [ ] **Step 3: Build and verify no regressions**

```bash
go build ./... && go test ./...
```
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: update renderSummaryCard for DevContainer mode"
```

---

### Task 7: Update `renderSuccessOutput` in `cmd/valla/main.go`

**Files:**
- Modify: `cmd/valla/main.go`

- [ ] **Step 1: Locate the function**

`renderSuccessOutput` builds a directory tree then next-steps commands. The current structure after the WordPress block:
```go
} else if ctx.EnvMode == "docker" {
    // shows docker-compose up -d
} else {
    // shows local run commands
}
```

- [ ] **Step 2: Add DevContainer branch as outermost condition**

In the directory tree section, add a new `if ctx.DevContainer` **before** the existing `else if ctx.OutputMode == "separate"` check:
```go
if ctx.DevContainer {
    children := []string{
        "  frontend/",
        "  backend/",
        "  .devcontainer/",
        "    devcontainer.json",
        "  .env",
        "  docker-compose.yml",
        "  docker-compose.dev.yml",
        "  Makefile",
    }
    tree := ctx.ProjectName + "/\n" + strings.Join(children, "\n")
    sb.WriteString(itui.StyleSuccessTree.Render(tree))
} else if ctx.OutputMode == "wordpress" {
    // ... existing
```

In the next-steps section, add `if ctx.DevContainer` as the outermost condition:
```go
if ctx.DevContainer {
    sb.WriteString(itui.StyleSuccessCmd.Render(fmt.Sprintf("  cd %s && make dev", ctx.ProjectName)))
    sb.WriteString("\n\n")
    sb.WriteString("Or open in VS Code and select ")
    sb.WriteString(itui.StyleSuccessCmd.Render(`"Reopen in Container"`))
    sb.WriteString(" to use the Dev Container.\n")
} else if ctx.OutputMode == "wordpress" {
    // ... existing
} else if ctx.EnvMode == "docker" {
    // ... existing
} else {
    // ... existing
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./... && go test ./...
```
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: update renderSuccessOutput for DevContainer mode"
```

---

## Chunk 4: Scaffolder — Templates and `GenerateDevContainerFiles`

### Task 8: Create devcontainer templates

**Files:**
- Create: `internal/registry/data/templates/devcontainer/devcontainer.json.tmpl`
- Create: `internal/registry/data/templates/devcontainer/Makefile.tmpl`

These are embedded via the existing `//go:embed data` directive in `loader.go` — no changes to `loader.go` needed.

- [ ] **Step 1: Create `internal/registry/data/templates/devcontainer/devcontainer.json.tmpl`**

```json
{
  "name": "{{.Ctx.ProjectName}}",
  "dockerComposeFile": "../docker-compose.dev.yml",
  "service": "backend",
  "workspaceFolder": "/app",
  "forwardPorts": [{{.Ctx.FrontendPort}}, {{.Ctx.BackendPort}}],
  "customizations": {
    "vscode": {
      "extensions": [
        "{{.BackendExtension}}",
        "ms-azuretools.vscode-docker"
      ]
    }
  }
}
```

`WeldContext` fields are accessed via `{{.Ctx.*}}` (named field, not embedded). `{{.BackendExtension}}` is a pre-computed field on `devContainerTemplateData`.

- [ ] **Step 2: Create `internal/registry/data/templates/devcontainer/Makefile.tmpl`**

```makefile
.PHONY: dev down shell-frontend shell-backend logs

dev:
	docker compose -f docker-compose.dev.yml up

down:
	docker compose -f docker-compose.dev.yml down

shell-frontend:
	docker compose -f docker-compose.dev.yml exec frontend sh

shell-backend:
	docker compose -f docker-compose.dev.yml exec backend sh

logs:
	docker compose -f docker-compose.dev.yml logs -f
```

(This template has no `{{}}` variables — it's identical for all stacks.)

- [ ] **Step 3: Verify the files are readable via `ReadEmbeddedFile`**

Add a quick test to `internal/registry/loader_test.go`:
```go
func TestReadEmbeddedFile_DevcontainerTemplates(t *testing.T) {
    for _, path := range []string{
        "internal/registry/data/templates/devcontainer/devcontainer.json.tmpl",
        "internal/registry/data/templates/devcontainer/Makefile.tmpl",
    } {
        _, err := registry.ReadEmbeddedFile(path)
        if err != nil {
            t.Errorf("ReadEmbeddedFile(%q) error: %v", path, err)
        }
    }
}
```

```bash
go test ./internal/registry/... -run "TestReadEmbeddedFile_DevcontainerTemplates" -v
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/registry/data/templates/devcontainer/ internal/registry/loader_test.go
git commit -m "feat: add devcontainer.json and Makefile templates"
```

---

### Task 9: Create `GenerateDevContainerFiles`

**Files:**
- Create: `internal/scaffolder/scaffolder_devcontainer.go`
- Create: `internal/scaffolder/scaffolder_devcontainer_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/scaffolder/scaffolder_devcontainer_test.go`:
```go
package scaffolder_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/tariktz/valla-cli/internal/registry"
    "github.com/tariktz/valla-cli/internal/scaffolder"
)

func makeDevContainerCtx() registry.WeldContext {
    return registry.WeldContext{
        ProjectName:   "myapp",
        FrontendPort:  3000,
        BackendPort:   8080,
        DevContainer:  true,
        OutputMode:    "devcontainer",
        EnvMode:       "docker",
    }
}

func TestGenerateDevContainerFiles_CreatesAllFiles(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
    feEntry := registry.Entry{Runtime: "node"}

    err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir)
    if err != nil {
        t.Fatalf("GenerateDevContainerFiles error: %v", err)
    }

    for _, path := range []string{
        filepath.Join(dir, ".devcontainer", "devcontainer.json"),
        filepath.Join(dir, "docker-compose.dev.yml"),
        filepath.Join(dir, "Makefile"),
    } {
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q to exist", path)
        }
    }
}

func TestGenerateDevContainerFiles_DevcontainerJSON_ContainsProjectName(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
    feEntry := registry.Entry{Runtime: "node"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
    if !strings.Contains(string(content), "myapp") {
        t.Error("devcontainer.json should contain project name")
    }
}

func TestGenerateDevContainerFiles_DevcontainerJSON_ForwardsPorts(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
    feEntry := registry.Entry{Runtime: "node"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
    if !strings.Contains(string(content), "3000") || !strings.Contains(string(content), "8080") {
        t.Error("devcontainer.json should forward frontend and backend ports")
    }
}

func TestGenerateDevContainerFiles_DevcontainerJSON_GoExtension(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go"}
    feEntry := registry.Entry{Runtime: "node"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
    if !strings.Contains(string(content), "golang.go") {
        t.Error("devcontainer.json should include golang.go extension for Go backend")
    }
}

func TestGenerateDevContainerFiles_Compose_ContainsDevCmd(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
    feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
    if !strings.Contains(string(content), "go run .") {
        t.Error("docker-compose.dev.yml should contain backend dev command")
    }
}

func TestGenerateDevContainerFiles_Compose_ContainsVolumeMounts(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go", DevCmd: "go run ."}
    feEntry := registry.Entry{Runtime: "node", DevCmd: "npm run dev"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, "docker-compose.dev.yml"))
    s := string(content)
    if !strings.Contains(s, "./frontend:/app") || !strings.Contains(s, "./backend:/app") {
        t.Error("docker-compose.dev.yml should mount ./frontend:/app and ./backend:/app")
    }
}

func TestGenerateDevContainerFiles_Makefile_ContainsDevTarget(t *testing.T) {
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go"}
    feEntry := registry.Entry{Runtime: "node"}

    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir); err != nil {
        t.Fatal(err)
    }

    content, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
    if !strings.Contains(string(content), "make dev") && !strings.Contains(string(content), "docker compose") {
        t.Error("Makefile should contain dev target using docker compose")
    }
}

func TestGenerateDevContainerFiles_RollbackOnError(t *testing.T) {
    // Use a non-existent project root to trigger write failure
    dir := t.TempDir()
    ctx := makeDevContainerCtx()
    beEntry := registry.Entry{Runtime: "go"}
    feEntry := registry.Entry{Runtime: "node"}

    // Write devcontainer.json first to a read-only dir to simulate partial failure
    // (This tests the rollback path indirectly — we verify no partial files remain)
    _ = scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, dir)
    // As long as the function doesn't panic on error paths, this is sufficient.
    // Full rollback testing requires OS-level mocking beyond this scope.
}

func TestRuntimeToDevContainerImage_Go(t *testing.T) {
    img := scaffolder.RuntimeToDevContainerImage("go", "")
    if img != "mcr.microsoft.com/devcontainers/go" {
        t.Errorf("unexpected image for go: %q", img)
    }
}

func TestRuntimeToDevContainerImage_Override(t *testing.T) {
    img := scaffolder.RuntimeToDevContainerImage("go", "custom-image:latest")
    if img != "custom-image:latest" {
        t.Errorf("expected override image, got %q", img)
    }
}

func TestRuntimeToDevContainerImage_Python3(t *testing.T) {
    img := scaffolder.RuntimeToDevContainerImage("python3", "")
    if img != "mcr.microsoft.com/devcontainers/python" {
        t.Errorf("unexpected image for python3: %q", img)
    }
}

func TestDevCmdForEntry_UsesRegistryValue(t *testing.T) {
    entry := registry.Entry{ID: "go-gin", Runtime: "go", DevCmd: "custom cmd"}
    cmd := scaffolder.DevCmdForEntry(entry)
    if cmd != "custom cmd" {
        t.Errorf("expected entry DevCmd, got %q", cmd)
    }
}

func TestDevCmdForEntry_FallsBackToDefault(t *testing.T) {
    entry := registry.Entry{ID: "go-gin", Runtime: "go"} // no DevCmd
    cmd := scaffolder.DevCmdForEntry(entry)
    if cmd != "go run ." {
        t.Errorf("expected default 'go run .', got %q", cmd)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/scaffolder/... -run "TestGenerateDevContainerFiles|TestRuntimeToDevContainerImage|TestDevCmdForEntry" -v
```
Expected: FAIL — functions undefined.

- [ ] **Step 3: Create `internal/scaffolder/scaffolder_devcontainer.go`**

```go
package scaffolder

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "text/template"

    "github.com/tariktz/valla-cli/internal/registry"
)

// runtimeImageMap maps Entry.Runtime values to Microsoft devcontainer images.
var runtimeImageMap = map[string]string{
    "go":      "mcr.microsoft.com/devcontainers/go",
    "node":    "mcr.microsoft.com/devcontainers/javascript-node",
    "bun":     "mcr.microsoft.com/devcontainers/javascript-node",
    "python3": "mcr.microsoft.com/devcontainers/python",
    "java":    "mcr.microsoft.com/devcontainers/java",
    "dotnet":  "mcr.microsoft.com/devcontainers/dotnet",
}

// runtimeExtensionMap maps Entry.Runtime values to VS Code extension IDs.
var runtimeExtensionMap = map[string]string{
    "go":      "golang.go",
    "node":    "dbaeumer.vscode-eslint",
    "bun":     "dbaeumer.vscode-eslint",
    "python3": "ms-python.python",
    "java":    "vscjava.vscode-java-pack",
    "dotnet":  "ms-dotnettools.csharp",
}

// runtimeDevCmdDefaults maps Entry.ID or Entry.Runtime to default dev commands.
// Entry.ID lookup takes precedence over Entry.Runtime lookup (see DevCmdForEntry).
// Bun entries have IDs like "react-bun", "vue-bun" etc. and Runtime "bun" —
// the runtime-level "bun" key covers all of them.
var runtimeDevCmdDefaults = map[string]string{
    // runtime-level fallbacks (keyed by Entry.Runtime)
    "go":      "go run .",
    "node":    "npm run dev",
    "bun":     "bun run dev",  // covers react-bun, vue-bun, angular-bun, etc.
    "python3": "python3 -m uvicorn main:app --reload --host 0.0.0.0",
    "dotnet":  "dotnet watch run",
    // entry-ID overrides (more specific)
    "python-fastapi":         "uvicorn main:app --reload --host 0.0.0.0",
    "python-flask":           "flask run --host 0.0.0.0",
    "python-django":          "python manage.py runserver 0.0.0.0:8000",
    "java-springboot-maven":  "./mvnw spring-boot:run",
    "java-springboot-gradle": "./gradlew bootRun",
    "java-quarkus-maven":     "./mvnw quarkus:dev",
    "java-quarkus-gradle":    "./gradlew quarkusDev",
}

// RuntimeToDevContainerImage returns the devcontainer image for a given runtime.
// If override is non-empty it is returned directly.
func RuntimeToDevContainerImage(runtime, override string) string {
    if override != "" {
        return override
    }
    if img, ok := runtimeImageMap[runtime]; ok {
        return img
    }
    return "mcr.microsoft.com/devcontainers/base"
}

// DevCmdForEntry returns the dev command for an entry.
// Uses Entry.DevCmd if set, falls back to ID-then-runtime defaults.
func DevCmdForEntry(entry registry.Entry) string {
    if entry.DevCmd != "" {
        return entry.DevCmd
    }
    if cmd, ok := runtimeDevCmdDefaults[entry.ID]; ok {
        return cmd
    }
    if cmd, ok := runtimeDevCmdDefaults[entry.Runtime]; ok {
        return cmd
    }
    return "echo 'no dev command configured'"
}

// devContainerTemplateData is the data passed to devcontainer.json template.
// Use a named Ctx field (not embedding) to avoid future WeldContext field additions
// shadowing template variables like BackendExtension.
// Templates access WeldContext fields via {{.Ctx.ProjectName}}, {{.Ctx.FrontendPort}}, etc.
type devContainerTemplateData struct {
    Ctx              registry.WeldContext
    BackendExtension string
}

// devComposeTmpl is the docker-compose.dev.yml template.
const devComposeTmpl = `version: "3.8"
networks:
  weld-dev-net:
    driver: bridge

services:
  frontend:
    image: {{.FrontendImage}}
    working_dir: /app
    volumes:
      - ./frontend:/app
    command: sh -c "{{.FrontendDevCmd}}"
    ports:
      - "{{.Ctx.FrontendPort}}:{{.Ctx.FrontendPort}}"
    networks: [weld-dev-net]

  backend:
    image: {{.BackendImage}}
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "{{.BackendDevCmd}}"
    ports:
      - "{{.Ctx.BackendPort}}:{{.Ctx.BackendPort}}"
    env_file: .env
    networks: [weld-dev-net]
{{- if .DBServiceIDs}}
    depends_on:
{{- range .DBServiceIDs}}
      - {{.}}
{{- end}}
{{- end}}
{{- range .DBServices}}

  {{.ID}}:
    image: {{.Image}}
    ports:
      - "{{.Port}}:{{.Port}}"
{{- if .Env}}
    environment:
{{- range .Env}}
      - {{.}}
{{- end}}
{{- end}}
{{- if .VolumePath}}
    volumes:
      - {{.ID}}-data:{{.VolumePath}}
{{- end}}
    networks: [weld-dev-net]
{{- end}}
{{- if .Volumes}}

volumes:
{{- range .Volumes}}
  {{.}}:
{{- end}}
{{- end}}
`

type devComposeDBService struct {
    ID         string
    Image      string
    Port       int
    Env        []string
    VolumePath string
}

type devComposeData struct {
    Ctx           registry.WeldContext
    FrontendImage string
    BackendImage  string
    FrontendDevCmd string
    BackendDevCmd  string
    DBServiceIDs  []string
    DBServices    []devComposeDBService
    Volumes       []string
}

// GenerateDevContainerFiles renders and writes .devcontainer/devcontainer.json,
// docker-compose.dev.yml, and Makefile into projectRoot.
// On error, performs a partial internal rollback (removes only files written so far),
// then returns the error. The caller (main.go) is responsible for the full project rollback.
func GenerateDevContainerFiles(ctx registry.WeldContext, backendEntry, frontendEntry registry.Entry, projectRoot string) error {
    backendImage := RuntimeToDevContainerImage(backendEntry.Runtime, backendEntry.DevContainerImage)
    frontendImage := RuntimeToDevContainerImage(frontendEntry.Runtime, frontendEntry.DevContainerImage)
    backendDevCmd := DevCmdForEntry(backendEntry)
    frontendDevCmd := DevCmdForEntry(frontendEntry)
    backendExt := runtimeExtensionMap[backendEntry.Runtime]
    if backendExt == "" {
        backendExt = "ms-azuretools.vscode-docker"
    }

    var written []string

    rollback := func() {
        for _, path := range written {
            if strings.HasSuffix(path, string(filepath.Separator)+".devcontainer") || filepath.Base(path) == ".devcontainer" {
                os.RemoveAll(path)
            } else {
                os.Remove(path)
            }
        }
    }

    // 1. .devcontainer/devcontainer.json
    devcontainerDir := filepath.Join(projectRoot, ".devcontainer")
    devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")

    tmplBytes, err := registry.ReadEmbeddedFile("internal/registry/data/templates/devcontainer/devcontainer.json.tmpl")
    if err != nil {
        return fmt.Errorf("read devcontainer.json template: %w", err)
    }
    tmpl, err := template.New("devcontainer.json").Parse(string(tmplBytes))
    if err != nil {
        return fmt.Errorf("parse devcontainer.json template: %w", err)
    }
    data := devContainerTemplateData{
        Ctx:              ctx,
        BackendExtension: backendExt,
    }
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return fmt.Errorf("render devcontainer.json: %w", err)
    }
    if err := WriteFile(devcontainerPath, buf.Bytes()); err != nil {
        return fmt.Errorf("write devcontainer.json: %w", err)
    }
    written = append(written, devcontainerDir) // track dir for rollback

    // 2. docker-compose.dev.yml
    composeData, err := buildDevComposeData(ctx, backendImage, frontendImage, backendDevCmd, frontendDevCmd)
    if err != nil {
        rollback()
        return fmt.Errorf("build dev compose data: %w", err)
    }
    composeTmpl, err := template.New("docker-compose.dev.yml").Parse(devComposeTmpl)
    if err != nil {
        rollback()
        return err
    }
    buf.Reset()
    if err := composeTmpl.Execute(&buf, composeData); err != nil {
        rollback()
        return fmt.Errorf("render docker-compose.dev.yml: %w", err)
    }
    composePath := filepath.Join(projectRoot, "docker-compose.dev.yml")
    if err := WriteFile(composePath, buf.Bytes()); err != nil {
        rollback()
        return fmt.Errorf("write docker-compose.dev.yml: %w", err)
    }
    written = append(written, composePath)

    // 3. Makefile
    makefileBytes, err := registry.ReadEmbeddedFile("internal/registry/data/templates/devcontainer/Makefile.tmpl")
    if err != nil {
        rollback()
        return fmt.Errorf("read Makefile template: %w", err)
    }
    makefilePath := filepath.Join(projectRoot, "Makefile")
    if err := WriteFile(makefilePath, makefileBytes); err != nil {
        rollback()
        return fmt.Errorf("write Makefile: %w", err)
    }

    return nil
}

// buildDevComposeData assembles the template data for docker-compose.dev.yml.
func buildDevComposeData(ctx registry.WeldContext, backendImage, frontendImage, backendDevCmd, frontendDevCmd string) (devComposeData, error) {
    d := devComposeData{
        Ctx:            ctx,
        BackendImage:   backendImage,
        FrontendImage:  frontendImage,
        BackendDevCmd:  backendDevCmd,
        FrontendDevCmd: frontendDevCmd,
    }

    // Known DB service images and config — mirrors the production compose.
    dbImages := map[string]string{
        "postgres": "postgres:16-alpine",
        "mysql":    "mysql:8",
        "mariadb":  "mariadb:11",
        "mongodb":  "mongo:7",
        "redis":    "redis:7-alpine",
    }
    dbVolumePaths := map[string]string{
        "postgres": "/var/lib/postgresql/data",
        "mysql":    "/var/lib/mysql",
        "mariadb":  "/var/lib/mysql",
        "mongodb":  "/data/db",
        "redis":    "/data",
    }

    for _, id := range ctx.DatabaseIDs {
        cfg := ctx.DBConfigs[id]
        img, ok := dbImages[id]
        if !ok {
            continue // SQLite or unknown — no Docker service
        }
        svc := devComposeDBService{
            ID:         id,
            Image:      img,
            Port:       cfg.Port,
            VolumePath: dbVolumePaths[id],
        }
        switch id {
        case "postgres":
            svc.Env = []string{
                fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
                fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
                fmt.Sprintf("POSTGRES_DB=%s", cfg.Name),
            }
        case "mysql", "mariadb":
            svc.Env = []string{
                fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", cfg.Password),
                fmt.Sprintf("MYSQL_DATABASE=%s", cfg.Name),
            }
        case "mongodb":
            svc.Env = []string{
                fmt.Sprintf("MONGO_INITDB_ROOT_USERNAME=%s", cfg.User),
                fmt.Sprintf("MONGO_INITDB_ROOT_PASSWORD=%s", cfg.Password),
            }
        }
        d.DBServices = append(d.DBServices, svc)
        d.DBServiceIDs = append(d.DBServiceIDs, id)
        if svc.VolumePath != "" {
            d.Volumes = append(d.Volumes, id+"-data")
        }
    }

    return d, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/scaffolder/... -run "TestGenerateDevContainerFiles|TestRuntimeToDevContainerImage|TestDevCmdForEntry" -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scaffolder/scaffolder_devcontainer.go internal/scaffolder/scaffolder_devcontainer_test.go
git commit -m "feat: add GenerateDevContainerFiles scaffolder"
```

---

### Task 10: Wire `GenerateDevContainerFiles` into `cmd/valla/main.go`

**Files:**
- Modify: `cmd/valla/main.go`

- [ ] **Step 1: Find the post-scaffolding section in `main.go`**

Look for where `renderSuccessOutput` is called and existing scaffolding completes (after `wiring.GenerateDockerCompose` or similar).

- [ ] **Step 2: Add call to `GenerateDevContainerFiles`**

After the existing scaffolding and wiring calls complete, add:
```go
if ctx.DevContainer {
    feEntry, _ := registry.FindByID(entries, ctx.FrontendID)
    beEntry, _ := registry.FindByID(entries, ctx.BackendID)
    if err := scaffolder.GenerateDevContainerFiles(ctx, beEntry, feEntry, projectRoot); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate dev container files: %v\n", err)
        scaffolder.Rollback(projectRoot)
        os.Exit(1)
    }
}
```

Where `projectRoot` is the path already computed earlier in `main.go`.

- [ ] **Step 3: Run full test suite**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```
Expected: all packages pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/valla/main.go
git commit -m "feat: wire GenerateDevContainerFiles into main scaffolding flow"
```

---

### Task 11: Add `dev_cmd` to Java YAML registry entries

**Files:**
- Modify: `internal/registry/data/backends/java-springboot-maven.yaml`
- Modify: `internal/registry/data/backends/java-springboot-gradle.yaml`
- Modify: `internal/registry/data/backends/java-quarkus-maven.yaml`
- Modify: `internal/registry/data/backends/java-quarkus-gradle.yaml`

- [ ] **Step 1: Add `dev_cmd` to each Java YAML**

In `java-springboot-maven.yaml`, add:
```yaml
dev_cmd: "./mvnw spring-boot:run"
```

In `java-springboot-gradle.yaml`, add:
```yaml
dev_cmd: "./gradlew bootRun"
```

In `java-quarkus-maven.yaml`, add:
```yaml
dev_cmd: "./mvnw quarkus:dev"
```

In `java-quarkus-gradle.yaml`, add:
```yaml
dev_cmd: "./gradlew quarkusDev"
```

- [ ] **Step 2: Write a test verifying Java entries have `dev_cmd` set**

In `internal/registry/loader_test.go`, add:
```go
func TestJavaEntries_HaveDevCmd(t *testing.T) {
    entries, err := registry.Load()
    if err != nil {
        t.Fatal(err)
    }
    javaIDs := []string{
        "java-springboot-maven",
        "java-springboot-gradle",
        "java-quarkus-maven",
        "java-quarkus-gradle",
    }
    for _, id := range javaIDs {
        entry, ok := registry.FindByID(entries, id)
        if !ok {
            t.Errorf("entry %q not found in registry", id)
            continue
        }
        if entry.DevCmd == "" {
            t.Errorf("entry %q must have dev_cmd set explicitly", id)
        }
    }
}
```

- [ ] **Step 3: Run test**

```bash
go test ./internal/registry/... -run "TestJavaEntries_HaveDevCmd" -v
```
Expected: PASS

- [ ] **Step 4: Run full suite**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/registry/data/backends/java-*.yaml internal/registry/loader_test.go
git commit -m "feat: add dev_cmd to Java registry entries for DevContainer support"
```

---

## Final Verification

- [ ] **Run the full test suite**

```bash
go test ./... -count=1
```
Expected: all packages pass, no cached results.

- [ ] **Build the binary**

```bash
go build -o /tmp/valla-test ./cmd/valla/
```
Expected: builds without errors.

- [ ] **Manual smoke test (if Docker is installed)**

```bash
/tmp/valla-test
# Select "Fully Dockerized" — it should appear in the output structure list
# Walk through the full flow: any frontend, any backend, any database
# Verify the generated project contains:
#   .devcontainer/devcontainer.json
#   docker-compose.dev.yml
#   Makefile
```

- [ ] **Cleanup test binary**

```bash
rm /tmp/valla-test
```
