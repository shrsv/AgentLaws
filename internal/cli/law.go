package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/lawedit"
	"github.com/athreyac4/agentlaws/internal/ordering"
	"github.com/athreyac4/agentlaws/internal/parser"
)

func newLawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "law",
		Short: "Manage individual numbered clauses within a section",
	}
	cmd.AddCommand(newLawAddCmd(), newLawListCmd(), newLawRemoveCmd())
	return cmd
}

// sectionFilePath resolves a section ID to its source file path by walking
// the book's ordering tree.
func sectionFilePath(book, id string) (string, error) {
	nodes, err := ordering.Tree(configPath(book))
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.ID == id {
			return filepath.Join(book, n.Path), nil
		}
	}
	return "", fmt.Errorf("%w: section %q", errNotFound, id)
}

func newLawAddCmd() *cobra.Command {
	var after int
	cmd := &cobra.Command{
		Use:   "add <book> <section-id> <text>",
		Short: "Append a new numbered clause to a section's laws",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, sectionID, text := args[0], args[1], args[2]
			path, err := sectionFilePath(book, sectionID)
			if err != nil {
				return err
			}
			if flagDryRun {
				cmd.Printf("would add clause to %s: %q\n", path, text)
				return nil
			}
			return lawedit.Add(path, text, after)
		},
	}
	cmd.Flags().IntVar(&after, "after", 0, "insert after this existing clause number")
	return cmd
}

func newLawListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <book> <section-id>",
		Short: "List a section's numbered clauses",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, sectionID := args[0], args[1]
			path, err := sectionFilePath(book, sectionID)
			if err != nil {
				return err
			}
			parsed, err := parser.ParseSection(path)
			if err != nil {
				return err
			}
			return printResult(cmd, parsed.RawLaws, func() {
				for i, law := range parsed.RawLaws {
					cmd.Printf("%d. %s\n", i+1, law.Text)
				}
			})
		},
	}
}

func newLawRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <book> <section-id> <number>",
		Short: "Remove a numbered clause from a section",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, sectionID, numStr := args[0], args[1], args[2]
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return &UsageError{Msg: fmt.Sprintf("invalid clause number %q", numStr)}
			}
			path, err := sectionFilePath(book, sectionID)
			if err != nil {
				return err
			}
			if flagDryRun {
				cmd.Printf("would remove clause %d from %s\n", n, path)
				return nil
			}
			return lawedit.Remove(path, n, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "remove without confirmation")
	return cmd
}
