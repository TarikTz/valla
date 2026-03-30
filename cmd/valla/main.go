package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tariktz/valla-cli/internal/detector"
	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/scaffolder"
	itui "github.com/tariktz/valla-cli/internal/tui"
	"github.com/tariktz/valla-cli/internal/tui/steps"
	"github.com/tariktz/valla-cli/internal/wiring"
)

var version = "dev"

func main() {
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

	entries, err := registry.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load registry: %v\n", err)
		os.Exit(1)
	}

	available := detector.Detect([]string{"go", "node", "bun", "python3", "dotnet"})
	// "java" is not a binary name so it cannot be in the Detect call above.
	// DetectWithAliases maps mvn/gradle presence to the logical "java" key.
	// This merge is safe because "java" is guaranteed absent from available above.
	javaAvailable := detector.DetectWithAliases(map[string][]string{
		"java": {"mvn", "gradle"},
	})
	for k, v := range javaAvailable {
		available[k] = v
	}
	feRuntimeOpts := []steps.RuntimeOption{
		{Name: "node", Available: available["node"], Reason: "node not found"},
		{Name: "bun", Available: available["bun"], Reason: "bun not found"},
	}
	beRuntimeOpts := []steps.RuntimeOption{
		{Name: "go", Available: available["go"], Reason: "go not found"},
		{Name: "node", Available: available["node"], Reason: "node not found"},
		{Name: "python3", Available: available["python3"], Reason: "python3 not found"},
		{Name: "dotnet", Available: available["dotnet"], Reason: "dotnet not found"},
		{Name: "java", Available: available["java"], Reason: "mvn or gradle not found"},
	}

	model := itui.New(entries, feRuntimeOpts, beRuntimeOpts)
	program := tea.NewProgram(model)
	startUpdateChecker(program, version)
	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	tuiModel := finalModel.(itui.Model)
	if !tuiModel.Confirmed() {
		os.Exit(0)
	}
	ctx := tuiModel.Context()
	fmt.Println()
	fmt.Println(renderSummaryCard(ctx, entries))
	fmt.Println()

	// WordPress always gets a dedicated root directory.
	if ctx.OutputMode == "wordpress" {
		projectRoot := ctx.ProjectName
		if err := os.MkdirAll(projectRoot, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create project dir: %v\n", err)
			os.Exit(1)
		}
		if err := generateWordPressProject(projectRoot, ctx); err != nil {
			_ = scaffolder.Rollback(projectRoot)
			fmt.Fprintf(os.Stderr, "WordPress scaffolding failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(renderSuccessOutput(ctx, "", "", registry.Entry{}, registry.Entry{}, ""))
		return
	}

	// Separate mode: services land directly in cwd as named siblings (no shared parent).
	// All other modes: services live inside a shared project root directory.
	isSeparate := ctx.OutputMode == "separate"
	frontendDir := "frontend"
	backendDir := "backend"
	if isSeparate {
		frontendDir = ctx.ProjectName + "-frontend"
		backendDir = ctx.ProjectName + "-backend"
	}

	var projectRoot string
	if !isSeparate {
		projectRoot = ctx.ProjectName
		if err := os.MkdirAll(projectRoot, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create project dir: %v\n", err)
			os.Exit(1)
		}
	}

	doRollback := func() {
		if isSeparate {
			_ = os.RemoveAll(frontendDir)
			_ = os.RemoveAll(backendDir)
		} else {
			_ = scaffolder.Rollback(projectRoot)
		}
	}

	var frontendEntry registry.Entry
	if ctx.FrontendID != "" {
		frontendEntry, _ = registry.FindByID(entries, ctx.FrontendID)
		printStage(fmt.Sprintf("Scaffolding frontend (%s)...", frontendEntry.Name))
		if err := runScaffold(ctx, frontendEntry, projectRoot, frontendDir, true); err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Frontend scaffolding failed: %v\n", err)
			os.Exit(1)
		}
	}

	var backendEntry registry.Entry
	if ctx.BackendID != "" {
		backendEntry, _ = registry.FindByID(entries, ctx.BackendID)
		printStage(fmt.Sprintf("Scaffolding backend (%s)...", backendEntry.Name))
		if err := runScaffold(ctx, backendEntry, projectRoot, backendDir, true); err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Backend scaffolding failed: %v\n", err)
			os.Exit(1)
		}
	}

	printStage("Wiring environment...")
	envContent := wiring.GenerateEnv(ctx)
	if err := os.WriteFile(filepath.Join(projectRoot, ".env"), []byte(envContent), 0o644); err != nil {
		doRollback()
		fmt.Fprintf(os.Stderr, "Failed to write .env: %v\n", err)
		os.Exit(1)
	}

	if ctx.EnvMode == "docker" {
		printStage("Generating Docker config...")
		var dbServices []wiring.DBServiceInput
		for _, id := range ctx.DatabaseIDs {
			entry, ok := registry.FindByID(entries, id)
			if !ok {
				continue
			}
			dbServices = append(dbServices, wiring.DBServiceInput{
				ID:     id,
				Docker: entry.Docker,
				Config: ctx.DBConfigs[id],
			})
		}
		composeContent, err := wiring.GenerateDockerCompose(wiring.DockerOptions{
			Ctx:      ctx,
			Frontend: frontendEntry.Docker,
			Backend:  backendEntry.Docker,
			DBs:      dbServices,
		})
		if err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Failed to generate docker-compose: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(projectRoot, "docker-compose.yml"), []byte(composeContent), 0o644); err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Failed to write docker-compose.yml: %v\n", err)
			os.Exit(1)
		}
	}

	if ctx.BackendID != "" && ctx.FrontendID != "" && backendEntry.CorsPatch != nil {
		corsFile := filepath.Join(projectRoot, backendDir, backendEntry.CorsPatch.File)
		source, err := os.ReadFile(corsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read %s for CORS patch: %v\n", corsFile, err)
		} else {
			injection, err := scaffolder.ApplyTemplate(backendEntry.CorsPatch.Template, ctx)
			if err == nil {
				patched, found, _ := wiring.ApplyCorsPatch(string(source), backendEntry.CorsPatch.Marker, injection)
				if !found {
					fmt.Fprintf(os.Stderr, "Warning: CORS marker not found in %s\n", corsFile)
				} else {
					_ = os.WriteFile(corsFile, []byte(patched), 0o644)
				}
			}
		}
	}

	if ctx.FrontendID != "" && frontendEntry.HTTPClientPatch != nil {
		apiURL := fmt.Sprintf("http://localhost:%d", ctx.BackendPort)
		if ctx.EnvMode == "docker" {
			apiURL = fmt.Sprintf("http://backend:%d", ctx.BackendPort)
		}
		clientContent := wiring.GenerateHTTPClientFile(apiURL)
		clientFile := filepath.Join(projectRoot, frontendDir, frontendEntry.HTTPClientPatch.File)
		if err := os.MkdirAll(filepath.Dir(clientFile), 0o755); err == nil {
			_ = os.WriteFile(clientFile, []byte(clientContent), 0o644)
		}
	}

	if ctx.ORMID != "" {
		printStage("Injecting ORM config...")
		var serviceDir string
		if ctx.BackendID != "" {
			serviceDir = filepath.Join(projectRoot, backendDir)
		} else {
			serviceDir = filepath.Join(projectRoot, frontendDir)
		}
		pdb := primarySQLDB(ctx.DatabaseIDs)
		switch ctx.ORMID {
		case "prisma":
			prismaProvider := "postgresql"
			if pdb == "mysql" || pdb == "mariadb" {
				prismaProvider = "mysql"
			}
			if pdb == "sqlite" {
				prismaProvider = "sqlite"
			}
			schemaContent, err := wiring.GeneratePrismaSchema(prismaProvider)
			if err == nil {
				schemaPath := filepath.Join(serviceDir, "prisma", "schema.prisma")
				if mkErr := os.MkdirAll(filepath.Dir(schemaPath), 0o755); mkErr == nil {
					_ = os.WriteFile(schemaPath, []byte(schemaContent), 0o644)
				}
				_ = os.WriteFile(filepath.Join(serviceDir, "prisma.config.ts"), []byte(wiring.GeneratePrismaConfig()), 0o644)
			}
		case "drizzle":
			dialect, importPath := "postgresql", "node-postgres"
			if pdb == "mysql" || pdb == "mariadb" {
				dialect, importPath = "mysql", "mysql2"
			}
			if pdb == "sqlite" {
				dialect, importPath = "sqlite", ""
			}
			cfgContent, idxContent, err := wiring.GenerateDrizzleConfig(dialect, importPath)
			if err == nil {
				_ = os.WriteFile(filepath.Join(serviceDir, "drizzle.config.ts"), []byte(cfgContent), 0o644)
				idxPath := filepath.Join(serviceDir, "src", "db", "index.ts")
				if mkErr := os.MkdirAll(filepath.Dir(idxPath), 0o755); mkErr == nil {
					_ = os.WriteFile(idxPath, []byte(idxContent), 0o644)
				}
			}
		}
	}

	clearStage()
	var ormInstr string
	if ctx.ORMID != "" {
		ormInstr = ormInstallInstructions(ctx.ORMID, primarySQLDB(ctx.DatabaseIDs))
	}
	fmt.Print(renderSuccessOutput(ctx, frontendDir, backendDir, frontendEntry, backendEntry, ormInstr))
}

// runScaffold handles one service: runs scaffold_cmd (or copies builtin_template),
// writes post_scaffold_files, and renames the output directory to targetDir.
func runScaffold(ctx registry.WeldContext, entry registry.Entry, root, targetDir string, quiet bool) error {
	tempName := scaffoldTempName(targetDir)
	ctx.ScaffoldName = tempName
	ctx.JavaArtifactID = strings.ReplaceAll(tempName, "-", "_")

	if entry.ScaffoldCmd != "" {
		rendered, err := scaffolder.ApplyTemplate(entry.ScaffoldCmd, ctx)
		if err != nil {
			return fmt.Errorf("rendering scaffold_cmd: %w", err)
		}
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", rendered)
		} else {
			cmd = exec.Command("sh", "-c", rendered)
		}
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "CI=1", "npm_config_yes=true")
		var stderrBuf strings.Builder
		if quiet {
			cmd.Stdout = nil
			cmd.Stderr = &stderrBuf
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			if quiet && stderrBuf.Len() > 0 {
				return fmt.Errorf("scaffold_cmd failed: %w\n%s", err, stderrBuf.String())
			}
			return fmt.Errorf("scaffold_cmd failed: %w", err)
		}
	} else if entry.BuiltinTemplate != "" {
		sourceFS, err := registry.ReadEmbeddedDir(entry.BuiltinTemplate)
		if err != nil {
			return fmt.Errorf("reading builtin_template: %w", err)
		}
		destinationDir := filepath.Join(root, tempName)
		if err := os.MkdirAll(destinationDir, 0o755); err != nil {
			return err
		}
		err = fs.WalkDir(sourceFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || path == "." {
				return err
			}
			destinationPath := filepath.Join(destinationDir, trimTemplateSuffix(path))
			if d.IsDir() {
				return os.MkdirAll(destinationPath, 0o755)
			}
			content, err := fs.ReadFile(sourceFS, path)
			if err != nil {
				return err
			}
			rendered, err := scaffolder.ApplyTemplate(string(content), ctx)
			if err != nil {
				return err
			}
			return scaffolder.WriteFile(destinationPath, []byte(rendered))
		})
		if err != nil {
			return fmt.Errorf("copying builtin_template: %w", err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, tempName)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("scaffolder did not create expected directory %q; the command may have prompted for input or written somewhere else", tempName)
		}
		return fmt.Errorf("checking scaffold output directory: %w", err)
	}

	for _, postScaffoldFile := range entry.PostScaffoldFiles {
		templateBytes, err := registry.ReadEmbeddedFile(postScaffoldFile.Template)
		if err != nil {
			return fmt.Errorf("reading post_scaffold_file template %s: %w", postScaffoldFile.Template, err)
		}
		rendered, err := scaffolder.ApplyTemplate(string(templateBytes), ctx)
		if err != nil {
			return fmt.Errorf("rendering post_scaffold_file %s: %w", postScaffoldFile.Path, err)
		}
		destinationPath := filepath.Join(root, tempName, postScaffoldFile.Path)
		if err := scaffolder.WriteFile(destinationPath, []byte(rendered)); err != nil {
			return fmt.Errorf("writing post_scaffold_file %s: %w", postScaffoldFile.Path, err)
		}
	}

	return scaffolder.RenameDir(filepath.Join(root, tempName), filepath.Join(root, targetDir))
}

func trimTemplateSuffix(path string) string {
	return strings.TrimSuffix(path, ".tmpl")
}

func scaffoldTempName(targetDir string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(targetDir) {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return "valla-tmp-svc"
	}
	return "valla-tmp-" + name
}

func generateWordPressProject(projectRoot string, ctx registry.WeldContext) error {
	envContent := wiring.GenerateWordPressEnv(ctx)
	if err := os.WriteFile(filepath.Join(projectRoot, ".env"), []byte(envContent), 0o644); err != nil {
		return err
	}
	if err := downloadWordPressFiles(filepath.Join(projectRoot, "wordpress")); err != nil {
		return err
	}
	if err := createWordPressTheme(filepath.Join(projectRoot, "wordpress"), wordpressThemeSlug(ctx.ProjectName), ctx.ProjectName); err != nil {
		return err
	}
	composeContent := wiring.GenerateWordPressCompose()
	if err := os.WriteFile(filepath.Join(projectRoot, "docker-compose.yml"), []byte(composeContent), 0o644); err != nil {
		return err
	}
	return nil
}

func downloadWordPressFiles(destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	response, err := http.Get("https://wordpress.org/latest.zip")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading wordpress: unexpected status %s", response.Status)
	}

	tempFile, err := os.CreateTemp("", "valla-wordpress-*.zip")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(tempFile, response.Body); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	archive, err := zip.OpenReader(tempPath)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, file := range archive.File {
		relativePath, ok := strings.CutPrefix(file.Name, "wordpress/")
		if !ok || relativePath == "" {
			continue
		}
		targetPath := filepath.Join(destination, filepath.FromSlash(relativePath))
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, filepath.Clean(destination)+string(os.PathSeparator)) && cleanTarget != filepath.Clean(destination) {
			return fmt.Errorf("invalid wordpress archive path: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			reader.Close()
			return err
		}
		if _, err := io.Copy(out, reader); err != nil {
			out.Close()
			reader.Close()
			return err
		}
		if err := out.Close(); err != nil {
			reader.Close()
			return err
		}
		if err := reader.Close(); err != nil {
			return err
		}
	}

	return nil
}

func createWordPressTheme(wordPressRoot, themeSlug, projectName string) error {
	themeDir := filepath.Join(wordPressRoot, "wp-content", "themes", themeSlug)
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		return err
	}
	styleCSS := fmt.Sprintf(`/*
Theme Name: %s
Text Domain: %s
Version: 0.1.0
*/

body {
	font-family: system-ui, sans-serif;
}
`, projectName, themeSlug)
	if err := os.WriteFile(filepath.Join(themeDir, "style.css"), []byte(styleCSS), 0o644); err != nil {
		return err
	}
	indexPHP := `<?php
get_header();
?>
<main>
  <h1><?php bloginfo('name'); ?></h1>
  <p>Custom theme is active.</p>
</main>
<?php
get_footer();
`
	if err := os.WriteFile(filepath.Join(themeDir, "index.php"), []byte(indexPHP), 0o644); err != nil {
		return err
	}
	functionsPHP := "<?php\nadd_action('wp_enqueue_scripts', function (): void {\n    wp_enqueue_style('" + themeSlug + "-style', get_stylesheet_uri(), [], null);\n});\n"
	if err := os.WriteFile(filepath.Join(themeDir, "functions.php"), []byte(functionsPHP), 0o644); err != nil {
		return err
	}
	return nil
}

// primarySQLDB returns the first SQL database ID from the slice, or "".
func primarySQLDB(ids []string) string {
	sqlDBs := map[string]bool{"postgres": true, "mysql": true, "mariadb": true, "sqlite": true}
	for _, id := range ids {
		if sqlDBs[id] {
			return id
		}
	}
	return ""
}

// ormInstallInstructions returns next-step install commands for the chosen ORM.
func ormInstallInstructions(ormID, primaryDBID string) string {
	if ormID == "prisma" {
		return "\nORM: Prisma\n  npm install prisma @prisma/client\n  npx prisma generate\n"
	}
	if ormID == "drizzle" {
		var runtime, dev string
		switch primaryDBID {
		case "mysql", "mariadb":
			runtime = "drizzle-orm mysql2 dotenv"
			dev = "drizzle-kit tsx"
		case "sqlite":
			runtime = "drizzle-orm better-sqlite3 dotenv"
			dev = "drizzle-kit tsx @types/better-sqlite3"
		default: // postgres
			runtime = "drizzle-orm pg dotenv"
			dev = "drizzle-kit tsx @types/pg"
		}
		return fmt.Sprintf("\nORM: Drizzle\n  npm install %s\n  npm install -D %s\n", runtime, dev)
	}
	return ""
}

// printStage prints a named scaffolding stage to stdout using a leading spinner char.
// Call it before each major step; it overwrites the previous line via \r.
func printStage(msg string) {
	fmt.Printf("\r\033[K⠸ %s", msg)
}

// clearStage clears the current spinner line.
func clearStage() {
	fmt.Print("\r\033[K")
}

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
		if len(ctx.DatabaseIDs) > 0 {
			ormLabel := "None"
			if ctx.ORMID == "prisma" {
				ormLabel = "Prisma"
			} else if ctx.ORMID == "drizzle" {
				ormLabel = "Drizzle"
			}
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

func wordpressThemeSlug(projectName string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(projectName) {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return "custom-theme"
	}
	return name
}
