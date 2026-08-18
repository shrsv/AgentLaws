package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

func newSectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "section",
		Short: "Manage sections (nested, level >= 2) within a book",
	}
	cmd.AddCommand(
		newSectionCreateCmd(),
		newSectionListCmd(),
		newSectionShowCmd(),
		newSectionMoveCmd(),
		newSectionRemoveCmd(),
	)
	return cmd
}

func newSectionCreateCmd() *cobra.Command {
	var bookFlag, title, id, parent, after, before string
	var position, level int
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create a new section under a parent chapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			p := alaws.Placement{After: after, Before: before, Position: position}
			if flagDryRun {
				cmd.Printf("would create %s/%s under %s and insert into %s\n", book, file, parent, configPath(book))
				return nil
			}
			if err := alaws.CreateSection(book, file, title, id, parent, level, p); err != nil {
				return err
			}
			cmd.Printf("created section %s (%s) under %s\n", id, file, parent)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&title, "title", "", "section title (required)")
	cmd.Flags().StringVar(&id, "id", "", "stable section ID (required)")
	cmd.Flags().StringVar(&parent, "parent", "", "parent chapter ID (required)")
	cmd.Flags().StringVar(&before, "before", "", "insert before this section/chapter ID")
	cmd.Flags().StringVar(&after, "after", "", "insert after this section/chapter ID")
	cmd.Flags().IntVar(&position, "position", 0, "insert at this 1-based position")
	cmd.Flags().IntVar(&level, "level", 0, "override the derived heading level")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("parent")
	return cmd
}

func newSectionListCmd() *cobra.Command {
	var chapter string
	cmd := &cobra.Command{
		Use:   "list [book]",
		Short: "List sections in a book, optionally filtered by parent chapter",
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
					if n.Level < 2 {
						continue
					}
					if chapter != "" && n.ParentID != chapter {
						continue
					}
					cmd.Printf("%s  (%s)  parent=%s\n", n.ID, n.Path, n.ParentID)
				}
			})
		},
	}
	cmd.Flags().StringVar(&chapter, "chapter", "", "only list sections under this chapter ID")
	return cmd
}

func newSectionShowCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a single section's metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			nodes, err := alaws.Tree(book)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				if n.ID == id {
					return printResult(cmd, n, func() {
						cmd.Printf("%s  (%s)  level=%d  parent=%s\n", n.ID, n.Path, n.Level, n.ParentID)
					})
				}
			}
			return fmt.Errorf("%w: section %q", errNotFound, id)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newSectionMoveCmd() *cobra.Command {
	var bookFlag, parent, before, after string
	var position int
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a section to a new parent and/or position",
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
			return alaws.MoveSection(book, id, parent, p)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&parent, "parent", "", "new parent chapter ID")
	cmd.Flags().StringVar(&before, "before", "", "move before this ID")
	cmd.Flags().StringVar(&after, "after", "", "move after this ID")
	cmd.Flags().IntVar(&position, "position", 0, "move to this 1-based position")
	return cmd
}

func newSectionRemoveCmd() *cobra.Command {
	var bookFlag string
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a section from a book",
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
			return alaws.RemoveSection(book, id, force)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().BoolVar(&force, "force", false, "remove even if the section has laws")
	return cmd
}
