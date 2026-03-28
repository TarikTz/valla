package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tariktz/valla-cli/internal/detector"
	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/scaffolder"
	itui "github.com/tariktz/valla-cli/internal/tui"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func main() {
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
	feRuntimes := detector.FilterByRuntime([]string{"node", "bun"}, available)
	beRuntimes := detector.FilterByRuntime([]string{"go", "node", "python3", "dotnet", "java"}, available)

	model := itui.New(entries, feRuntimes, beRuntimes)
	program := tea.NewProgram(model)
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
		fmt.Printf("\nWordPress project scaffolded successfully.\n\nNext steps:\n  cd %s\n  docker-compose up -d\n\n", ctx.ProjectName)
		fmt.Printf("Open http://localhost:%d and finish the WordPress setup in the browser.\n", ctx.FrontendPort)
		fmt.Printf("MySQL is preconfigured with DB=%s user=%s password=%s on host db:%d.\n", ctx.DBName, ctx.DBUser, ctx.DBPassword, ctx.DBPort)
		fmt.Printf("Develop themes locally in wordpress/wp-content/themes/%s.\n", wordpressThemeSlug(ctx.ProjectName))
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
		fmt.Printf("Scaffolding frontend (%s)...\n", frontendEntry.Name)
		if err := runScaffold(ctx, frontendEntry, projectRoot, frontendDir); err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Frontend scaffolding failed: %v\n", err)
			os.Exit(1)
		}
	}

	var backendEntry registry.Entry
	if ctx.BackendID != "" {
		backendEntry, _ = registry.FindByID(entries, ctx.BackendID)
		fmt.Printf("Scaffolding backend (%s)...\n", backendEntry.Name)
		if err := runScaffold(ctx, backendEntry, projectRoot, backendDir); err != nil {
			doRollback()
			fmt.Fprintf(os.Stderr, "Backend scaffolding failed: %v\n", err)
			os.Exit(1)
		}
	}

	databaseEntry, _ := registry.FindByID(entries, ctx.DatabaseID)
	isSQLite := databaseEntry.SQLite

	fmt.Println("Writing .env...")
	envContent := wiring.GenerateEnv(ctx, isSQLite)
	if err := os.WriteFile(filepath.Join(projectRoot, ".env"), []byte(envContent), 0o644); err != nil {
		doRollback()
		fmt.Fprintf(os.Stderr, "Failed to write .env: %v\n", err)
		os.Exit(1)
	}

	if ctx.EnvMode == "docker" {
		fmt.Println("Writing docker-compose.yml...")
		var dbDocker *registry.DockerConfig
		if databaseEntry.Docker != nil {
			dbDocker = databaseEntry.Docker
		}
		composeContent, err := wiring.GenerateDockerCompose(wiring.DockerOptions{
			Ctx:      ctx,
			Frontend: frontendEntry.Docker,
			Backend:  backendEntry.Docker,
			DB:       dbDocker,
			IsSQLite: isSQLite,
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
		fmt.Println("Configuring CORS...")
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
		fmt.Println("Configuring HTTP client...")
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

	fmt.Printf("\nProject scaffolded successfully.\n\n")
	if ctx.EnvMode == "docker" {
		if isSeparate {
			fmt.Println("Next steps:")
			fmt.Println("  docker-compose up -d")
		} else {
			fmt.Printf("Next steps:\n  cd %s\n  docker-compose up -d\n", ctx.ProjectName)
		}
	} else {
		if isSeparate {
			fmt.Println("Next steps:")
			if ctx.FrontendID != "" {
				fmt.Printf("  cd %s && npm install\n", frontendDir)
			}
			if ctx.BackendID != "" {
				if backendEntry.Runtime == "go" {
					fmt.Printf("  cd %s && go run main.go\n", backendDir)
				} else if backendEntry.Runtime == "python3" {
					fmt.Printf("  cd %s && source venv/bin/activate && python ...\n", backendDir)
				} else {
					fmt.Printf("  cd %s && npm install && npm start\n", backendDir)
				}
			}
		} else {
			fmt.Printf("Next steps:\n  cd %s\n\n", ctx.ProjectName)
			if ctx.FrontendID != "" {
				fmt.Printf("  npm install    (in /%s)\n", frontendDir)
			}
			if ctx.BackendID != "" {
				if backendEntry.Runtime == "go" {
					fmt.Printf("  go run main.go (in /%s)\n", backendDir)
				} else if backendEntry.Runtime == "python3" {
					fmt.Printf("  source venv/bin/activate && python ... (in /%s)\n", backendDir)
				} else {
					fmt.Printf("  npm install && npm start (in /%s)\n", backendDir)
				}
			}
		}
	}
}

// runScaffold handles one service: runs scaffold_cmd (or copies builtin_template),
// writes post_scaffold_files, and renames the output directory to targetDir.
func runScaffold(ctx registry.WeldContext, entry registry.Entry, root, targetDir string) error {
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
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
