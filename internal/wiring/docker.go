package wiring

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/tariktz/valla-cli/internal/registry"
)

// DBServiceInput bundles everything needed to render one database service
// in docker-compose.yml.
type DBServiceInput struct {
	ID     string                 // registry ID: "postgres", "redis", etc.
	Docker *registry.DockerConfig // nil for SQLite (no Docker service)
	Config registry.DBConfig      // resolved credentials / port / path
}

// DockerOptions groups all inputs needed to generate docker-compose.yml.
type DockerOptions struct {
	Ctx      registry.WeldContext
	Frontend *registry.DockerConfig // nil when no frontend selected
	Backend  *registry.DockerConfig // nil when no backend selected
	DBs      []DBServiceInput       // one entry per selected database; empty = none
}

const composeTmpl = `version: "3.8"
networks:
  weld-net:
    driver: bridge
{{- if .Volumes}}
volumes:
{{- range .Volumes}}
  {{.}}:
{{- end}}
{{- end}}

services:
{{- if .Frontend}}
  frontend:
    build:
      context: {{.FrontendContext}}
      dockerfile: {{.Frontend.Dockerfile}}
    ports:
      - "{{.Ctx.FrontendPort}}:{{.Ctx.FrontendPort}}"
{{- if .FrontendEnv}}
    environment:
{{- range .FrontendEnv}}
      - {{.}}
{{- end}}
{{- end}}
    networks: [weld-net]
{{- end}}
{{- if .Backend}}

  backend:
    build:
      context: {{.BackendContext}}
      dockerfile: {{.Backend.Dockerfile}}
    ports:
      - "{{.Ctx.BackendPort}}:{{.Ctx.BackendPort}}"
{{- if .BackendEnv}}
    environment:
{{- range .BackendEnv}}
      - {{.}}
{{- end}}
{{- end}}
    env_file: .env
    networks: [weld-net]
{{- if .DBServiceIDs}}
    depends_on:
{{- range .DBServiceIDs}}
      - {{.}}
{{- end}}
{{- end}}
{{- end}}
{{- range .DBServices}}

  {{.ID}}:
    image: {{.Docker.Image}}
    ports:
      - "{{.Config.Port}}:{{.Config.Port}}"
{{- if .Env}}
    environment:
{{- range .Env}}
      {{.}}
{{- end}}
{{- end}}
{{- if .Docker.VolumePath}}
    volumes:
      - {{.ID}}-data:{{.Docker.VolumePath}}
{{- end}}
    networks: [weld-net]
{{- end}}
`

// composeDBService is the template-ready representation of one DB service.
type composeDBService struct {
	ID     string
	Docker *registry.DockerConfig
	Config registry.DBConfig
	Env    []string
}

type composeData struct {
	DockerOptions
	FrontendContext string
	BackendContext  string
	FrontendEnv     []string
	BackendEnv      []string
	DBServices      []composeDBService
	DBServiceIDs    []string
	Volumes         []string
}

// GenerateDockerCompose produces the content of docker-compose.yml.
func GenerateDockerCompose(opts DockerOptions) (string, error) {
	var frontendContext, backendContext string
	if opts.Frontend != nil {
		frontendContext = opts.Frontend.BuildContext
		if opts.Ctx.OutputMode == "separate" {
			frontendContext = fmt.Sprintf("./%s-frontend", opts.Ctx.ProjectName)
		}
	}
	if opts.Backend != nil {
		backendContext = opts.Backend.BuildContext
		if opts.Ctx.OutputMode == "separate" {
			backendContext = fmt.Sprintf("./%s-backend", opts.Ctx.ProjectName)
		}
	}

	var err error
	var frontendEnv []string
	if opts.Frontend != nil {
		frontendEnv, err = renderDockerEnv(opts.Frontend.EnvVars, opts.Ctx, false)
		if err != nil {
			return "", err
		}
	}
	var backendEnv []string
	if opts.Backend != nil {
		backendEnv, err = renderDockerEnv(opts.Backend.EnvVars, opts.Ctx, false)
		if err != nil {
			return "", err
		}
	}

	var dbServices []composeDBService
	var dbServiceIDs []string
	var volumes []string
	for _, db := range opts.DBs {
		if db.Docker == nil {
			// SQLite or other file-based DB: no Docker service
			continue
		}
		dbEnv, err := renderDockerEnv(db.Docker.EnvVars, db.Config, true)
		if err != nil {
			return "", err
		}
		dbServices = append(dbServices, composeDBService{
			ID:     db.ID,
			Docker: db.Docker,
			Config: db.Config,
			Env:    dbEnv,
		})
		dbServiceIDs = append(dbServiceIDs, db.ID)
		if db.Docker.VolumePath != "" {
			volumes = append(volumes, db.ID+"-data")
		}
	}

	tmpl, err := template.New("compose").Parse(composeTmpl)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, composeData{
		DockerOptions:   opts,
		FrontendContext: frontendContext,
		BackendContext:  backendContext,
		FrontendEnv:     frontendEnv,
		BackendEnv:      backendEnv,
		DBServices:      dbServices,
		DBServiceIDs:    dbServiceIDs,
		Volumes:         volumes,
	}); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// renderDockerEnv renders templated env var values for a Docker service.
// data is passed directly as the template context: pass registry.WeldContext
// for frontend/backend services, and registry.DBConfig for DB services.
func renderDockerEnv(envVars map[string]string, data any, useComposeSyntax bool) ([]string, error) {
	if len(envVars) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		tmpl, err := template.New("docker-env").Parse(envVars[key])
		if err != nil {
			return nil, err
		}
		var buffer bytes.Buffer
		if err := tmpl.Execute(&buffer, data); err != nil {
			return nil, err
		}
		if useComposeSyntax {
			out = append(out, fmt.Sprintf("%s: %s", key, buffer.String()))
		} else {
			out = append(out, fmt.Sprintf("%s=%s", key, buffer.String()))
		}
	}
	return out, nil
}
