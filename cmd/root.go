// Package cmd implements the rren command-line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/giulianozor/rren/internal/config"
	"github.com/giulianozor/rren/internal/renamer"
)

var (
	flagDryRun    bool
	flagConfigFile string
	flagRecursive bool
	flagYes       bool
	flagPath      string
)

var rootCmd = &cobra.Command{
	Use:   "rren",
	Short: "Rename files and folders using regular expressions",
	Long: `rren renames files and folders by applying a chain of regular expressions
defined in a YAML configuration file.

Each rule in the configuration file can target files, folders, or both.
A preview table is shown before any changes are made, and confirmation is
required unless the --yes flag is provided.`,
	RunE: run,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&flagDryRun, "dry-run", "n", false, "Preview renames without applying any changes")
	rootCmd.Flags().StringVarP(&flagConfigFile, "config", "c", config.DefaultConfigPath(), "Path to the configuration file")
	rootCmd.Flags().BoolVarP(&flagRecursive, "recursive", "r", false, "Rename files and folders recursively")
	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip the confirmation prompt")
	rootCmd.Flags().StringVarP(&flagPath, "path", "p", ".", "Root path to process")
}

func run(cmd *cobra.Command, args []string) error {
	// Load config.
	cfg, err := config.Load(flagConfigFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if len(cfg.Rules) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no rules defined in config file")
		return nil
	}

	// Build the rename plan.
	entries, err := renamer.Plan(flagPath, cfg, flagRecursive)
	if err != nil {
		return fmt.Errorf("building rename plan: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println("No files or folders found.")
		return nil
	}

	// Print the preview table.
	fmt.Println()
	renamer.PrintTable(entries)

	// Count how many would actually be renamed.
	toRename := 0
	for _, e := range entries {
		if !e.Skipped {
			toRename++
		}
	}

	if toRename == 0 {
		fmt.Println("\nNo renames to perform.")
		return nil
	}

	// Confirm unless --yes or --dry-run.
	if !flagYes && !flagDryRun {
		if !renamer.Confirm(fmt.Sprintf("\nProceed with %d rename(s)?", toRename)) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Execute renames (no-op in dry-run mode).
	results := renamer.Execute(entries, flagDryRun)

	// Print summary.
	renamer.PrintSummaryFull(entries, results, flagDryRun)

	return nil
}
