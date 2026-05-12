package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/giulianozor/rren/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rren.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeConfig(t, `
rules:
  - description: "TV episodes"
    pattern: '.*(\d\d)[exEX](\d+).*\.(\w+)'
    replacement: 'S${1}E${2}.${3}'
    apply_to: files
  - pattern: '(.*)\s\((\d{4})\).*(\.m..)'
    replacement: '$2 - $1$3'
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(cfg.Rules))
	}

	r0 := cfg.Rules[0]
	if r0.ApplyTo != config.ApplyToFiles {
		t.Errorf("rule 0 apply_to: want %q, got %q", config.ApplyToFiles, r0.ApplyTo)
	}
	if r0.Description != "TV episodes" {
		t.Errorf("rule 0 description: want %q, got %q", "TV episodes", r0.Description)
	}

	// Rule without apply_to should default to "both".
	r1 := cfg.Rules[1]
	if r1.ApplyTo != config.ApplyToBoth {
		t.Errorf("rule 1 apply_to: want %q, got %q", config.ApplyToBoth, r1.ApplyTo)
	}
}

func TestLoad_MissingPattern(t *testing.T) {
	path := writeConfig(t, `
rules:
  - replacement: 'foo'
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing pattern, got nil")
	}
}

func TestLoad_MissingReplacement(t *testing.T) {
	path := writeConfig(t, `
rules:
  - pattern: 'foo'
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing replacement, got nil")
	}
}

func TestLoad_InvalidApplyTo(t *testing.T) {
	path := writeConfig(t, `
rules:
  - pattern: 'foo'
    replacement: 'bar'
    apply_to: everything
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid apply_to, got nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/rren.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
