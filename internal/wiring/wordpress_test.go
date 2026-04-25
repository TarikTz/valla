package wiring_test

import (
	"strings"
	"testing"

	"github.com/tariktz/valla-cli/internal/registry"
	"github.com/tariktz/valla-cli/internal/wiring"
)

func TestGenerateWordPressEnv(t *testing.T) {
	ctx := registry.WeldContext{
		FrontendPort: 8080,
		DBName:       "my_blog",
		DatabaseIDs:  []string{"mysql"},
		DBConfigs: map[string]registry.DBConfig{
			"mysql": {
				Port:     8051,
				Host:     "db",
				User:     "wordpress",
				Password: "wordpress",
				Name:     "my_blog",
			},
		},
	}
	out := wiring.GenerateWordPressEnv(ctx)
	for _, expected := range []string{
		"WORDPRESS_PORT=8080",
		"MYSQL_PORT=8051",
		"MYSQL_DATABASE=my_blog",
		"WORDPRESS_DB_HOST=db:3306",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in env output", expected)
		}
	}
}

func TestGenerateWordPressCompose(t *testing.T) {
	out := wiring.GenerateWordPressCompose()
	for _, expected := range []string{"wordpress:", "db:", "./wordpress:/var/www/html", "mysql-data:"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in compose output", expected)
		}
	}
}
