package scaffolder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
var runtimeDevCmdDefaults = map[string]string{
	// runtime-level fallbacks (keyed by Entry.Runtime)
	"go":      "go run .",
	"node":    "npm run dev",
	"bun":     "bun run dev",
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

// depVolume describes a named volume that isolates a runtime's dependency
// directory inside the container, preventing the bind mount from overwriting
// installed packages.
type depVolume struct {
	Name string // named volume name, e.g. "backend_node_modules"
	Path string // container path to shadow, e.g. "/app/node_modules"
}

// runtimeDepVolume returns the dep-isolation volume for the given service and
// runtime, or nil if the runtime does not need one.
//
// Only runtimes that install packages inside /app (the bind-mounted path)
// need a named volume to shadow that sub-directory. Runtimes like go, java,
// and dotnet store caches outside /app (e.g. /root/.m2) so they are
// unaffected by the bind mount and do not require a shadow volume.
func runtimeDepVolume(service, runtime string) *depVolume {
	switch runtime {
	case "node", "bun":
		return &depVolume{Name: service + "_node_modules", Path: "/app/node_modules"}
	case "python3":
		return &depVolume{Name: service + "_venv", Path: "/app/.venv"}
	}
	return nil
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
{{- if .FrontendDepVolume}}
      - {{.FrontendDepVolume.Name}}:{{.FrontendDepVolume.Path}}
{{- end}}
    command: sh -c "{{.FrontendDevCmd}}"
    ports:
      - "{{.Ctx.FrontendPort}}:{{.Ctx.FrontendPort}}"
    networks: [weld-dev-net]

  backend:
    image: {{.BackendImage}}
    working_dir: /app
    volumes:
      - ./backend:/app
{{- if .BackendDepVolume}}
      - {{.BackendDepVolume.Name}}:{{.BackendDepVolume.Path}}
{{- end}}
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
	Ctx               registry.WeldContext
	FrontendImage     string
	BackendImage      string
	FrontendDevCmd    string
	BackendDevCmd     string
	DBServiceIDs      []string
	DBServices        []devComposeDBService
	Volumes           []string
	FrontendDepVolume *depVolume
	BackendDepVolume  *depVolume
}

// GenerateDevContainerFiles renders and writes .devcontainer/devcontainer.json,
// docker-compose.dev.yml, and Makefile into projectRoot.
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
			if filepath.Base(path) == ".devcontainer" {
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
	written = append(written, devcontainerDir)

	// 2. docker-compose.dev.yml
	composeData, err := buildDevComposeData(ctx, backendImage, frontendImage, backendDevCmd, frontendDevCmd, frontendEntry.Runtime, backendEntry.Runtime)
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
	written = append(written, makefilePath)

	// 4. .gitignore
	gitignoreBytes, err := registry.ReadEmbeddedFile("internal/registry/data/templates/devcontainer/gitignore.tmpl")
	if err != nil {
		rollback()
		return fmt.Errorf("read .gitignore template: %w", err)
	}
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	if err := WriteFile(gitignorePath, gitignoreBytes); err != nil {
		rollback()
		return fmt.Errorf("write .gitignore: %w", err)
	}

	return nil
}

// buildDevComposeData assembles the template data for docker-compose.dev.yml.
func buildDevComposeData(ctx registry.WeldContext, backendImage, frontendImage, backendDevCmd, frontendDevCmd, frontendRuntime, backendRuntime string) (devComposeData, error) {
	feDepVol := runtimeDepVolume("frontend", frontendRuntime)
	beDepVol := runtimeDepVolume("backend", backendRuntime)
	d := devComposeData{
		Ctx:               ctx,
		BackendImage:      backendImage,
		FrontendImage:     frontendImage,
		BackendDevCmd:     backendDevCmd,
		FrontendDevCmd:    frontendDevCmd,
		FrontendDepVolume: feDepVol,
		BackendDepVolume:  beDepVol,
	}
	if feDepVol != nil {
		d.Volumes = append(d.Volumes, feDepVol.Name)
	}
	if beDepVol != nil {
		d.Volumes = append(d.Volumes, beDepVol.Name)
	}

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
