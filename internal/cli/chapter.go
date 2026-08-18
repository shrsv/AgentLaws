package cli

import (
	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func newChapterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chapter",
		Short: "Manage chapters (top-level sections) within a book",
	}
	cmd.AddCommand(
		newChapterCreateCmd(),
		newChapterListCmd(),
		newChapterMoveCmd(),
		newChapterRemoveCmd(),
	)
	return cmd
}

func newChapterCreateCmd() *cobra.Command {
	var bookFlag, title, id, after, before string
	var position int
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create a new chapter (a level-1 section) in a book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			p := alaws.Placement{After: after, Before: before, Position: position}
			if flagDryRun {
				cmd.Printf("would create %s/%s and insert into %s\n", book, file, configPath(book))
				return nil
			}
			if err := alaws.CreateChapter(book, file, title, id, p); err != nil {
				return err
			}
			cmd.Printf("created chapter %s (%s)\n", id, file)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&title, "title", "", "chapter title (required)")
	cmd.Flags().StringVar(&id, "id", "", "stable section ID (required)")
	cmd.Flags().StringVar(&before, "before", "", "insert before this chapter/section ID")
	cmd.Flags().StringVar(&after, "after", "", "insert after this chapter/section ID")
	cmd.Flags().IntVar(&position, "position", 0, "insert at this 1-based position")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("id")
	return cmd
}

func newChapterListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [book]",
		Short: "List chapters in a book",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}
			nodes, err := alaws.Tree(book)
			if err != nil {
				return err
			}
			return printResult(cmd, nodes, func() {
				for _, n := range nodes {
					if n.Level == 1 {
						cmd.Printf("%s  (%s)\n", n.ID, n.Path)
					}
				}
			})
		},
	}
}

func newChapterMoveCmd() *cobra.Command {
	var bookFlag, before, after string
	var position int
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a chapter to a new position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			p := alaws.Placement{After: after, Before: before, Position: position}
			if flagDryRun {
				cmd.Printf("would move %s in %s\n", id, configPath(book))
				return nil
			}
			return alaws.MoveChapter(book, id, p)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&before, "before", "", "move before this chapter ID")
	cmd.Flags().StringVar(&after, "after", "", "move after this chapter ID")
	cmd.Flags().IntVar(&position, "position", 0, "move to this 1-based position")
	return cmd
}

func newChapterRemoveCmd() *cobra.Command {
	var bookFlag string
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a chapter from a book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			if flagDryRun {
				cmd.Printf("would remove %s from %s\n", id, configPath(book))
				return nil
			}
			return alaws.RemoveChapter(book, id, force)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().BoolVar(&force, "force", false, "remove even if the chapter has sections under it")
	return cmd
}

// firstArg returns args[0], or "" if args is empty - used by commands
// whose sole positional is an optional book path.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
