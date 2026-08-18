package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/ordering"
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
	var title, id, after string
	var position int
	cmd := &cobra.Command{
		Use:   "create <book> <file>",
		Short: "Create a new chapter (a level-1 section) in a book",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, file := args[0], args[1]
			path := filepath.Join(book, file)
			meta := ordering.SectionMeta{Title: title, ID: id, Level: 1}
			if flagDryRun {
				cmd.Printf("would create %s and insert into %s\n", path, configPath(book))
				return nil
			}
			if err := ordering.NewSectionFile(path, meta); err != nil {
				return err
			}
			if err := ordering.Insert(configPath(book), file, placement(after, position)); err != nil {
				return err
			}
			cmd.Printf("created chapter %s (%s)\n", id, file)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "chapter title (required)")
	cmd.Flags().StringVar(&id, "id", "", "stable section ID (required)")
	cmd.Flags().StringVar(&after, "after", "", "insert after this chapter/section ID")
	cmd.Flags().IntVar(&position, "position", 0, "insert at this 1-based position")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("id")
	return cmd
}

func newChapterListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <book>",
		Short: "List chapters in a book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := ordering.Tree(configPath(args[0]))
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
	var before, after string
	var position int
	cmd := &cobra.Command{
		Use:   "move <book> <id>",
		Short: "Move a chapter to a new position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			p := placement(after, position)
			if before != "" {
				// Placement is expressed as "after"; moving before X means
				// after X's predecessor - resolved once ordering.Tree exists.
				p.After = before
			}
			if flagDryRun {
				cmd.Printf("would move %s in %s\n", id, configPath(book))
				return nil
			}
			return ordering.Move(configPath(book), id, p)
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "move before this chapter ID")
	cmd.Flags().StringVar(&after, "after", "", "move after this chapter ID")
	cmd.Flags().IntVar(&position, "position", 0, "move to this 1-based position")
	return cmd
}

func newChapterRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <book> <id>",
		Short: "Remove a chapter from a book",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			if flagDryRun {
				cmd.Printf("would remove %s from %s\n", id, configPath(book))
				return nil
			}
			return ordering.Remove(configPath(book), id, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "remove even if the chapter has sections under it")
	return cmd
}

// placement builds an ordering.Placement from the CLI's --after/--position
// flags (books.go and section.go build the equivalent for their own flags).
func placement(after string, position int) ordering.Placement {
	return ordering.Placement{After: after, Position: position}
}
