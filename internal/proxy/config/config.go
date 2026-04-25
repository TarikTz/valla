package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Service represents one proxied upstream in valla.yaml.
type Service struct {
	Name      string `yaml:"name"`
	Port      int    `yaml:"port"`
	Subdomain string `yaml:"subdomain"`
}

// Config is the parsed representation of a valla.yaml file.
type Config struct {
	Project  string    `yaml:"project"`
	Domain   string    `yaml:"domain"`
	Services []Service `yaml:"services"`
}

// Load reads and validates a valla.yaml file at the given path.
// It returns (nil, nil) when the file does not exist, so callers can
// treat a missing file as "no config" without an error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Project == "" {
		return errors.New("project field is required")
	}
	if len(cfg.Services) == 0 {
		return errors.New("at least one service is required")
	}
	for i, svc := range cfg.Services {
		if svc.Name == "" {
			return fmt.Errorf("services[%d]: name is required", i)
		}
		if svc.Port <= 0 || svc.Port > 65535 {
			return fmt.Errorf("services[%d] (%s): invalid port %d", i, svc.Name, svc.Port)
		}
		if svc.Subdomain == "" {
			return fmt.Errorf("services[%d] (%s): subdomain is required", i, svc.Name)
		}
	}
	return nil
}
