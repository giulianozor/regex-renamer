// Package renamer implements the core file/folder rename logic for rren.
package renamer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/giulianozor/rren/internal/config"
)

// Entry represents a single file or folder that may be renamed.
type Entry struct {
	Dir     string // parent directory
	OldName string // original base name
	NewName string // computed new base name (may equal OldName if skipped)
	IsDir   bool
	Depth   int    // directory depth from the root path
	Skipped bool   // true when no rule produced a new name
}

// OldPath returns the full original path of the entry.
func (e *Entry) OldPath() string { return filepath.Join(e.Dir, e.OldName) }

// NewPath returns the full destination path of the entry.
func (e *Entry) NewPath() string { return filepath.Join(e.Dir, e.NewName) }

// Result holds the outcome of a single rename operation.
type Result struct {
	Entry *Entry
	Err   error
}

// Plan builds the list of entries that would be renamed under root.
// When recursive is false only the immediate children of root are included.
func Plan(root string, cfg *config.Config, recursive bool) ([]*Entry, error) {
	// Compile all patterns once.
	compiled := make([]*regexp.Regexp, len(cfg.Rules))
	for i, rule := range cfg.Rules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid pattern %q: %w", i+1, rule.Pattern, err)
		}
		compiled[i] = re
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", root, err)
	}

	var entries []*Entry

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root itself.
		if path == absRoot {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		depth := strings.Count(rel, string(os.PathSeparator))

		// When not recursive, skip entries deeper than depth 0.
		if !recursive && depth > 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		dir := filepath.Dir(path)
		oldName := d.Name()

		newName := applyRules(oldName, d.IsDir(), cfg.Rules, compiled)
		skipped := newName == oldName

		entries = append(entries, &Entry{
			Dir:     dir,
			OldName: oldName,
			NewName: newName,
			IsDir:   d.IsDir(),
			Depth:   depth,
			Skipped: skipped,
		})
		return nil
	}

	if err := filepath.WalkDir(absRoot, walkFn); err != nil {
		return nil, fmt.Errorf("walking %q: %w", absRoot, err)
	}

	return entries, nil
}

// applyRules chains all matching rules on name and returns the final name.
func applyRules(name string, isDir bool, rules []config.Rule, compiled []*regexp.Regexp) string {
	for i, rule := range rules {
		switch rule.ApplyTo {
		case config.ApplyToFiles:
			if isDir {
				continue
			}
		case config.ApplyToFolders:
			if !isDir {
				continue
			}
		}
		name = compiled[i].ReplaceAllString(name, rule.Replacement)
	}
	return name
}

// Execute renames entries that are not skipped.
// Entries are renamed deepest-first to avoid path invalidation when parent
// directories are also being renamed.
// Returns the list of results (one per non-skipped entry).
func Execute(entries []*Entry, dryRun bool) []Result {
	// Sort deepest first.
	sorted := make([]*Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Depth > sorted[j].Depth
	})

	var results []Result
	for _, e := range sorted {
		if e.Skipped {
			continue
		}
		var err error
		if !dryRun {
			err = os.Rename(e.OldPath(), e.NewPath())
		}
		results = append(results, Result{Entry: e, Err: err})
	}
	return results
}

// PrintTable prints a formatted table of all planned renames to stdout.
func PrintTable(entries []*Entry) {
	const (
		colType = "Type"
		colSrc  = "Source"
		colDst  = "Destination"
		skipped = "-- Skipped --"
	)

	// Compute column widths.
	typeW := len(colType)
	srcW := len(colSrc)
	dstW := len(colDst)
	for _, e := range entries {
		t := entryType(e)
		if len(t) > typeW {
			typeW = len(t)
		}
		if len(e.OldName) > srcW {
			srcW = len(e.OldName)
		}
		dst := e.NewName
		if e.Skipped {
			dst = skipped
		}
		if len(dst) > dstW {
			dstW = len(dst)
		}
	}

	sep := fmt.Sprintf("+-%-*s-+-%-*s-+-%-*s-+", typeW, strings.Repeat("-", typeW), srcW, strings.Repeat("-", srcW), dstW, strings.Repeat("-", dstW))
	hdr := fmt.Sprintf("| %-*s | %-*s | %-*s |", typeW, colType, srcW, colSrc, dstW, colDst)

	fmt.Println(sep)
	fmt.Println(hdr)
	fmt.Println(sep)
	for _, e := range entries {
		dst := e.NewName
		if e.Skipped {
			dst = skipped
		}
		fmt.Printf("| %-*s | %-*s | %-*s |\n", typeW, entryType(e), srcW, e.OldName, dstW, dst)
	}
	fmt.Println(sep)
}

// PrintSummaryFull prints the full rename summary including skipped count.
func PrintSummaryFull(entries []*Entry, results []Result, dryRun bool) {
	var renamed, errors, skipped int
	for _, e := range entries {
		if e.Skipped {
			skipped++
		}
	}
	for _, r := range results {
		if r.Err != nil {
			errors++
		} else {
			renamed++
		}
	}

	fmt.Println()
	if dryRun {
		fmt.Println("=== Dry-run summary (no files were changed) ===")
	} else {
		fmt.Println("=== Summary ===")
	}
	fmt.Printf("  Renamed : %d\n", renamed)
	fmt.Printf("  Skipped : %d\n", skipped)
	if errors > 0 {
		fmt.Printf("  Errors  : %d\n", errors)
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("    ✗ %s → %s: %v\n", r.Entry.OldName, r.Entry.NewName, r.Err)
			}
		}
	}
}

// Confirm asks the user for a yes/no confirmation and returns true on "y" or "yes".
func Confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}

func entryType(e *Entry) string {
	if e.IsDir {
		return "dir"
	}
	return "file"
}
