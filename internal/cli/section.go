package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/ordering"
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
	var title, id, parent, after string
	var position, level int
	cmd := &cobra.Command{
		Use:   "create <book> <file>",
		Short: "Create a new section under a parent chapter",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, file := args[0], args[1]
			path := filepath.Join(book, file)

			explicitLevel := level > 0
			resolvedLevel := level
			if !explicitLevel {
				nodes, err := ordering.Tree(configPath(book))
				if err != nil {
					return err
				}
				for _, n := range nodes {
					if n.ID == parent {
						resolvedLevel = n.Level + 1
						break
					}
				}
				if resolvedLevel == 0 {
					return fmt.Errorf("parent %q not found in %s", parent, configPath(book))
				}
			}

			// Only write an explicit level into the file when the caller
			// asked for one directly, or when the file's own folder depth
			// wouldn't already produce it by default (see levelOverride).
			metaLevel := resolvedLevel
			if !explicitLevel {
				metaLevel = levelOverride(file, resolvedLevel)
			}
			meta := ordering.SectionMeta{Title: title, ID: id, Level: metaLevel}
			if flagDryRun {
				cmd.Printf("would create %s (level %d, parent %s) and insert into %s\n", path, resolvedLevel, parent, configPath(book))
				return nil
			}
			if err := ordering.NewSectionFile(path, meta); err != nil {
				return err
			}
			p := placement(after, position)
			if p.After == "" && p.Position == 0 {
				// Default: insert as the parent's last descendant.
				p.After = parent
			}
			if err := ordering.Insert(configPath(book), file, p); err != nil {
				return err
			}
			cmd.Printf("created section %s (%s) under %s\n", id, file, parent)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "section title (required)")
	cmd.Flags().StringVar(&id, "id", "", "stable section ID (required)")
	cmd.Flags().StringVar(&parent, "parent", "", "parent chapter ID (required)")
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
		Use:   "list <book>",
		Short: "List sections in a book, optionally filtered by parent chapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := ordering.Tree(configPath(args[0]))
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
	return &cobra.Command{
		Use:   "show <book> <id>",
		Short: "Show a single section's metadata",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := ordering.Tree(configPath(args[0]))
			if err != nil {
				return err
			}
			id := args[1]
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
}

func newSectionMoveCmd() *cobra.Command {
	var parent, before, after string
	var position int
	cmd := &cobra.Command{
		Use:   "move <book> <id>",
		Short: "Move a section to a new parent and/or position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, id := args[0], args[1]
			p := ordering.Placement{After: after, Before: before, Position: position}
			if parent != "" && p.After == "" && p.Before == "" && p.Position == 0 {
				// Default: append as the new parent's last child.
				p.After = parent
			}
			if flagDryRun {
				cmd.Printf("would move %s in %s\n", id, configPath(book))
				return nil
			}
			if err := ordering.Move(configPath(book), id, p); err != nil {
				return err
			}
			if parent == "" {
				return nil
			}
			// Moving under a new parent changes this section's intended
			// nesting depth. The file didn't move, so fix its frontmatter
			// to match (an explicit override if its folder depth no
			// longer implies the right level, cleared otherwise).
			nodes, err := ordering.Tree(configPath(book))
			if err != nil {
				return err
			}
			var childPath string
			parentLevel := -1
			for _, n := range nodes {
				if n.ID == id {
					childPath = n.Path
				}
				if n.ID == parent {
					parentLevel = n.Level
				}
			}
			if childPath == "" {
				return fmt.Errorf("%w: section %q", errNotFound, id)
			}
			if parentLevel == -1 {
				return fmt.Errorf("%w: chapter/section %q", errNotFound, parent)
			}
			return ordering.SetLevel(filepath.Join(book, childPath), levelOverride(childPath, parentLevel+1))
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "new parent chapter ID")
	cmd.Flags().StringVar(&before, "before", "", "move before this ID")
	cmd.Flags().StringVar(&after, "after", "", "move after this ID")
	cmd.Flags().IntVar(&position, "position", 0, "move to this 1-based position")
	return cmd
}

func newSectionRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <book> <id>",
		Short: "Remove a section from a book",
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
	cmd.Flags().BoolVar(&force, "force", false, "remove even if the section has laws")
	return cmd
}
