package registry

// DBConfig holds per-database connection credentials and addressing.
// Keyed by registry ID in WeldContext.DBConfigs.
// Invariant: SQLite is true if and only if Path is non-empty.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string // schema/database name; empty for Redis and MongoDB
	Path     string // SQLite only — non-empty signals file-based DB
	SQLite   bool   // true for SQLite; drives env/docker branching logic
}

// WeldContext holds all user choices and is used for template rendering throughout.
type WeldContext struct {
	ProjectName    string              // final project name (step 1)
	ScaffoldName   string              // temporary name passed to scaffold_cmd (e.g. "valla-tmp-frontend")
	JavaArtifactID string              // ScaffoldName with hyphens replaced by underscores (valid Java artifact ID)
	FrontendID     string              // selected frontend registry entry ID (e.g. "react")
	BackendID      string              // selected backend registry entry ID (e.g. "go-gin")
	DatabaseIDs    []string            // selected database registry IDs (multi-select)
	DBConfigs      map[string]DBConfig // per-DB credentials keyed by registry ID
	FrontendPort   int
	BackendPort    int
	DBName         string // default schema/database name for all SQL DBs (derived from ProjectName)
	OutputMode     string // "monorepo", "separate", or "wordpress"
	EnvMode        string // "local" or "docker"
}

// DockerConfig describes how a service is containerized.
type DockerConfig struct {
	Image        string            `yaml:"image"`
	BuildContext string            `yaml:"build_context"`
	Dockerfile   string            `yaml:"dockerfile"`
	EnvVars      map[string]string `yaml:"env_vars"`
	VolumePath   string            `yaml:"volume_path"` // container-side volume mount path; empty = no volume
}

// CorsPatch describes how to inject CORS config into a backend file.
type CorsPatch struct {
	File     string `yaml:"file"`
	Marker   string `yaml:"marker"`
	Template string `yaml:"template"`
}

// HTTPClientPatch describes how to inject the API base URL into a frontend file.
type HTTPClientPatch struct {
	File     string `yaml:"file"`
	Mode     string `yaml:"mode"` // "create" or "inject"
	Template string `yaml:"template"`
}

// PostScaffoldFile describes a file valla-cli writes after running scaffold_cmd.
type PostScaffoldFile struct {
	Path     string `yaml:"path"`
	Template string `yaml:"template"` // path to embedded template file
}

// Entry is a single registry entry describing one stack variant.
type Entry struct {
	ID                string             `yaml:"id"`
	Name              string             `yaml:"name"`
	Type              string             `yaml:"type"`    // "frontend", "backend", "database"
	Runtime           string             `yaml:"runtime"` // detected binary name (e.g. "go", "node")
	Group             string             `yaml:"group"`   // display group for two-step selection
	RequiresRuntime   string             `yaml:"requires_runtime"`
	ScaffoldCmd       string             `yaml:"scaffold_cmd"`     // empty = use BuiltinTemplate
	BuiltinTemplate   string             `yaml:"builtin_template"` // path inside registry/templates/
	DefaultPort       int                `yaml:"default_port"`
	DBPathDefault     string             `yaml:"db_path_default"` // SQLite only
	SQLite            bool               `yaml:"sqlite"`
	EnvVars           []string           `yaml:"env_vars"`
	CorsPatch         *CorsPatch         `yaml:"cors_patch"`
	HTTPClientPatch   *HTTPClientPatch   `yaml:"http_client_patch"`
	PostScaffoldFiles []PostScaffoldFile `yaml:"post_scaffold_files"`
	Docker            *DockerConfig      `yaml:"docker"`
}
