package renamer_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/giulianozor/rren/internal/config"
	"github.com/giulianozor/rren/internal/renamer"
)

// buildCfg is a test helper that creates a Config with the given rules,
// verifying patterns compile.
func buildCfg(t *testing.T, rules []config.Rule) *config.Config {
	t.Helper()
	for i, r := range rules {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			t.Fatalf("rule %d bad pattern: %v", i, err)
		}
		if rules[i].ApplyTo == "" {
			rules[i].ApplyTo = config.ApplyToBoth
		}
	}
	return &config.Config{Rules: rules}
}

// makeTempDir creates a temporary directory with the given file names.
func makeTempDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatalf("creating %q: %v", path, err)
		}
	}
	return dir
}

func TestPlan_TVEpisode(t *testing.T) {
	dir := makeTempDir(t,
		"show.name.01x02.720p.mkv",
		"another.show.02E05.1080p.mp4",
		"not_an_episode.txt",
	)
	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `.*(\d\d)[exEX](\d+).*\.(\w+)`,
			Replacement: `S${1}E${2}.${3}`,
			ApplyTo:     config.ApplyToFiles,
		},
	})

	entries, err := renamer.Plan(dir, cfg, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}

	want := map[string]string{
		"show.name.01x02.720p.mkv":    "S01E02.mkv",
		"another.show.02E05.1080p.mp4": "S02E05.mp4",
		"not_an_episode.txt":           "not_an_episode.txt", // skipped
	}
	for _, e := range entries {
		exp, ok := want[e.OldName]
		if !ok {
			t.Errorf("unexpected entry %q", e.OldName)
			continue
		}
		if e.NewName != exp {
			t.Errorf("%q: want NewName %q, got %q", e.OldName, exp, e.NewName)
		}
		if e.OldName == "not_an_episode.txt" && !e.Skipped {
			t.Errorf("%q should be marked as skipped", e.OldName)
		}
	}
}

func TestPlan_MovieRename(t *testing.T) {
	dir := makeTempDir(t,
		"Inception (2010) BluRay.mkv",
		"The Dark Knight (2008) 1080p.mp4",
		"notes.txt",
	)
	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `(.*)\s\((\d{4})\).*(\.m..)`,
			Replacement: `$2 - $1$3`,
			ApplyTo:     config.ApplyToFiles,
		},
	})

	entries, err := renamer.Plan(dir, cfg, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	want := map[string]string{
		"Inception (2010) BluRay.mkv":      "2010 - Inception.mkv",
		"The Dark Knight (2008) 1080p.mp4": "2008 - The Dark Knight.mp4",
		"notes.txt":                         "notes.txt",
	}
	for _, e := range entries {
		exp, ok := want[e.OldName]
		if !ok {
			t.Errorf("unexpected entry %q", e.OldName)
			continue
		}
		if e.NewName != exp {
			t.Errorf("%q: want NewName %q, got %q", e.OldName, exp, e.NewName)
		}
	}
}

func TestPlan_ApplyToFolders(t *testing.T) {
	dir := t.TempDir()
	// Create a sub-directory and a file.
	if err := os.Mkdir(filepath.Join(dir, "show.name.01x02"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "show.name.01x02.mkv"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `.*(\d\d)[exEX](\d+).*`,
			Replacement: `S${1}E${2}`,
			ApplyTo:     config.ApplyToFolders,
		},
	})

	entries, err := renamer.Plan(dir, cfg, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	for _, e := range entries {
		if e.IsDir {
			if e.NewName != "S01E02" {
				t.Errorf("dir: want NewName %q, got %q", "S01E02", e.NewName)
			}
		} else {
			// File should be skipped because apply_to is folders.
			if !e.Skipped {
				t.Errorf("file %q should be skipped when apply_to=folders", e.OldName)
			}
		}
	}
}

func TestExecute_DryRun(t *testing.T) {
	dir := makeTempDir(t, "show.01x02.mkv")
	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `.*(\d\d)[exEX](\d+).*\.(\w+)`,
			Replacement: `S${1}E${2}.${3}`,
			ApplyTo:     config.ApplyToFiles,
		},
	})

	entries, err := renamer.Plan(dir, cfg, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	results := renamer.Execute(entries, true /* dryRun */)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("unexpected error in dry run: %v", results[0].Err)
	}

	// File must still have the original name (dry run = no changes).
	if _, err := os.Stat(filepath.Join(dir, "show.01x02.mkv")); os.IsNotExist(err) {
		t.Error("original file must still exist after dry run")
	}
}

func TestExecute_RealRename(t *testing.T) {
	dir := makeTempDir(t, "show.01x02.mkv")
	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `.*(\d\d)[exEX](\d+).*\.(\w+)`,
			Replacement: `S${1}E${2}.${3}`,
			ApplyTo:     config.ApplyToFiles,
		},
	})

	entries, err := renamer.Plan(dir, cfg, false)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	results := renamer.Execute(entries, false /* dryRun */)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("rename error: %v", results[0].Err)
	}

	if _, err := os.Stat(filepath.Join(dir, "S01E02.mkv")); os.IsNotExist(err) {
		t.Error("renamed file S01E02.mkv should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "show.01x02.mkv")); !os.IsNotExist(err) {
		t.Error("original file should no longer exist after rename")
	}
}

func TestPlan_Recursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "show.01x02.mkv"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	cfg := buildCfg(t, []config.Rule{
		{
			Pattern:     `.*(\d\d)[exEX](\d+).*\.(\w+)`,
			Replacement: `S${1}E${2}.${3}`,
			ApplyTo:     config.ApplyToFiles,
		},
	})

	entriesFlat, _ := renamer.Plan(dir, cfg, false)
	entriesRecursive, _ := renamer.Plan(dir, cfg, true)

	// Non-recursive: only the sub-directory itself (file inside is deeper).
	fileInFlat := false
	for _, e := range entriesFlat {
		if e.OldName == "show.01x02.mkv" {
			fileInFlat = true
		}
	}
	if fileInFlat {
		t.Error("non-recursive: should not find file inside sub-directory")
	}

	// Recursive: file inside sub-dir must be found.
	fileInRecursive := false
	for _, e := range entriesRecursive {
		if e.OldName == "show.01x02.mkv" {
			fileInRecursive = true
		}
	}
	if !fileInRecursive {
		t.Error("recursive: file inside sub-directory should be found")
	}
}
