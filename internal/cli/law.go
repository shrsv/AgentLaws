package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

func newLawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "law",
		Short: "Manage individual numbered clauses within a section",
	}
	cmd.AddCommand(newLawAddCmd(), newLawListCmd(), newLawRemoveCmd())
	return cmd
}

func newLawAddCmd() *cobra.Command {
	var bookFlag string
	var after int
	cmd := &cobra.Command{
		Use:   "add <section-id> <text>",
		Short: "Append a new numbered clause to a section's laws",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID, text := args[0], args[1]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			if flagDryRun {
				path, err := alaws.SectionFilePath(book, sectionID)
				if err != nil {
					return err
				}
				cmd.Printf("would add clause to %s: %q\n", path, text)
				return nil
			}
			return alaws.AddLaw(book, sectionID, text, after)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().IntVar(&after, "after", 0, "insert after this existing clause number")
	return cmd
}

func newLawListCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "list <section-id>",
		Short: "List a section's numbered clauses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			laws, err := alaws.ListLaws(book, sectionID)
			if err != nil {
				return err
			}
			return printResult(cmd, laws, func() {
				for i, text := range laws {
					cmd.Printf("%d. %s\n", i+1, text)
				}
			})
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newLawRemoveCmd() *cobra.Command {
	var bookFlag string
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <section-id> <number>",
		Short: "Remove a numbered clause from a section",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID, numStr := args[0], args[1]
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return &UsageError{Msg: fmt.Sprintf("invalid clause number %q", numStr)}
			}
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			if flagDryRun {
				path, err := alaws.SectionFilePath(book, sectionID)
				if err != nil {
					return err
				}
				cmd.Printf("would remove clause %d from %s\n", n, path)
				return nil
			}
			return alaws.RemoveLaw(book, sectionID, n, force)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().BoolVar(&force, "force", false, "remove without confirmation")
	return cmd
}
