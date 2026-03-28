package wiring

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/tariktz/valla-cli/internal/registry"
)

// DockerOptions groups all inputs needed to generate docker-compose.yml.
type DockerOptions struct {
	Ctx      registry.WeldContext
	Frontend *registry.DockerConfig // nil when no frontend selected
	Backend  *registry.DockerConfig // nil when no backend selected
	DB       *registry.DockerConfig
	IsSQLite bool
}

const composeTmpl = `version: "3.8"
networks:
  weld-net:
    driver: bridge
{{- if and .DB (not .IsSQLite)}}
volumes:
  db-data:
{{- end}}

services:
{{- if .Frontend}}
  frontend:
    build:
      context: {{.FrontendContext}}
      dockerfile: {{.Frontend.Dockerfile}}
    ports:
      - "{{.Ctx.FrontendPort}}:{{.Ctx.FrontendPort}}"
{{- if .FrontendEnv }}
    environment:
{{- range .FrontendEnv }}
      - {{ . }}
{{- end }}
{{- end }}
    networks: [weld-net]
{{- if .Backend}}
    depends_on: [backend]
{{- end}}
{{- end}}
{{- if .Backend}}

  backend:
    build:
      context: {{.BackendContext}}
      dockerfile: {{.Backend.Dockerfile}}
    ports:
      - "{{.Ctx.BackendPort}}:{{.Ctx.BackendPort}}"
{{- if .BackendEnv }}
    environment:
{{- range .BackendEnv }}
      - {{ . }}
{{- end }}
{{- end }}
    env_file: .env
    networks: [weld-net]
{{- if and .DB (not .IsSQLite)}}
    depends_on: [db]
{{- end}}
{{- end}}
{{- if and .DB (not .IsSQLite)}}

  db:
    image: {{.DB.Image}}
    ports:
      - "{{.Ctx.DBPort}}:{{.Ctx.DBPort}}"
{{- if .DBEnv }}
    environment:
{{- range .DBEnv }}
      {{ . }}
{{- end }}
{{- end }}
    volumes:
      - db-data:/var/lib/postgresql/data
    networks: [weld-net]
{{- end}}
`

type composeData struct {
	DockerOptions
	FrontendContext string
	BackendContext  string
	FrontendEnv     []string
	BackendEnv      []string
	DBEnv           []string
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
	var dbEnv []string
	if opts.DB != nil {
		dbEnv, err = renderDockerEnv(opts.DB.EnvVars, opts.Ctx, true)
		if err != nil {
			return "", err
		}
	}

	tmpl, err := template.New("compose").Funcs(template.FuncMap{
		"not": func(value bool) bool { return !value },
	}).Parse(composeTmpl)
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
		DBEnv:           dbEnv,
	}); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func renderDockerEnv(envVars map[string]string, ctx registry.WeldContext, useComposeSyntax bool) ([]string, error) {
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
		if err := tmpl.Execute(&buffer, ctx); err != nil {
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
