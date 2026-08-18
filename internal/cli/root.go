// Package cli implements the alaws command-line interface described in
// docs/PLAN1.md §32. Every command is a thin wrapper over the internal/ and
// pkg/alaws libraries: no command contains logic that doesn't also live in
// the library (PLAN1 §52), so the CLI, the Go API, and the future UI stay
// behaviorally identical.
package cli

import (
	"github.com/spf13/cobra"
)

// Global flags shared across subcommands (PLAN1 §32 "Cross-cutting
// behavior").
var (
	flagRoot   string
	flagJSON   bool
	flagDryRun bool
)

// Exit codes, per PLAN1 §32: 0 success, 1 validation/compile error,
// 2 usage error, 3 not found.
const (
	ExitOK       = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitNotFound = 3
)

// Execute runs the alaws root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "alaws",
		Short: "Govern AI agents through prompts organized like law",
		Long: `AgentLaws (alaws) compiles a lawbook of Markdown sections into a
deterministic, citable, provenance-tracked body of law for AI agents.

Run 'alaws <command> --help' for details on any command.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagRoot, "root", ".", "book root to search when a book path is not given explicitly")
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "machine-readable output, for read commands")
	root.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "preview a mutating command's changes without writing them")

	root.AddCommand(
		newInitCmd(),
		newBooksCmd(),
		newChapterCmd(),
		newSectionCmd(),
		newLawCmd(),
		newCompileCmd(),
		newExportCmd(),
		newValidateCmd(),
		newListCmd(),
		newShowCmd(),
		newResolveCmd(),
		newHistoryCmd(),
		newRenderCmd(),
		newWatchCmd(),
		newServeCmd(),
		newUICmd(),
		newSignCmd(),
		newVerifyCmd(),
	)

	return root
}
