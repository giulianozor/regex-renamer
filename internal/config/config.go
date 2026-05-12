// Package config handles loading and validating rren configuration files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ApplyTo controls whether a rule is applied to files, folders, or both.
type ApplyTo string

const (
	ApplyToFiles   ApplyTo = "files"
	ApplyToFolders ApplyTo = "folders"
	ApplyToBoth    ApplyTo = "both"
)

// Rule represents a single rename rule.
type Rule struct {
	Description string  `yaml:"description"`
	Pattern     string  `yaml:"pattern"`
	Replacement string  `yaml:"replacement"`
	ApplyTo     ApplyTo `yaml:"apply_to"`
}

// Config holds the full rren configuration.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// DefaultConfigPath returns the default config file path (~/.config/rren.yaml).
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.config/rren.yaml"
	}
	return filepath.Join(home, ".config", "rren.yaml")
}

// Load reads and parses a YAML config file from the given path.
// It expands a leading ~ to the user's home directory.
func Load(path string) (*Config, error) {
	path = expandHome(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Normalise each rule.
	for i, r := range cfg.Rules {
		if r.Pattern == "" {
			return nil, fmt.Errorf("rule %d: missing required field 'pattern'", i+1)
		}
		if r.Replacement == "" {
			return nil, fmt.Errorf("rule %d: missing required field 'replacement'", i+1)
		}
		switch r.ApplyTo {
		case ApplyToFiles, ApplyToFolders, ApplyToBoth:
			// valid
		case "":
			cfg.Rules[i].ApplyTo = ApplyToBoth
		default:
			return nil, fmt.Errorf("rule %d: invalid apply_to value %q (must be files, folders, or both)", i+1, r.ApplyTo)
		}
	}

	return &cfg, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
