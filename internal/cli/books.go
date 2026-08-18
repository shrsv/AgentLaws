package cli

import (
	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/discovery"
	"github.com/athreyac4/agentlaws/internal/ordering"
	"github.com/athreyac4/agentlaws/internal/parser"
)

func newInitCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a new book (alias for `books create`)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			return runBooksCreate(cmd, path, title)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "title of the new book")
	return cmd
}

func newBooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "books",
		Short: "Manage lawbook clusters (books)",
	}
	cmd.AddCommand(newBooksListCmd(), newBooksCreateCmd(), newBooksShowCmd())
	return cmd
}

func newBooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Discover all books (alaws.toml clusters) under --root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusters, err := discovery.FindClusters(flagRoot)
			if err != nil {
				return err
			}
			return printResult(cmd, clusters, func() {
				for _, c := range clusters {
					title := c.Title
					if title == "" {
						title = "(untitled)"
					}
					cmd.Printf("%s  %s\n", c.Path, title)
				}
			})
		},
	}
}

func newBooksCreateCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Create a new book at path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBooksCreate(cmd, args[0], title)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "title of the new book")
	return cmd
}

func runBooksCreate(cmd *cobra.Command, path string, title string) error {
	if flagDryRun {
		cmd.Printf("would create %s/alaws.toml with title %q\n", path, title)
		return nil
	}
	if err := ordering.NewBook(path, title); err != nil {
		return err
	}
	cmd.Printf("created %s/alaws.toml\n", path)
	return nil
}

// BookInfo is the JSON/human shape of `alaws books show`: the book's own
// title (PLAN1 §4) alongside its ordering tree.
type BookInfo struct {
	Title    string          `json:"title"`
	Sections []ordering.Node `json:"sections"`
}

func newBooksShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <path>",
		Short: "Show a book's title, ordering tree, and metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := configPath(args[0])
			meta, err := parser.ParseLawbookConfig(cfgPath)
			if err != nil {
				return err
			}
			nodes, err := ordering.Tree(cfgPath)
			if err != nil {
				return err
			}
			info := BookInfo{Title: meta.Title, Sections: nodes}
			return printResult(cmd, info, func() {
				cmd.Printf("%s  (%s)\n", meta.Title, args[0])
				for _, n := range nodes {
					cmd.Printf("  level %d  %s  (%s)\n", n.Level, n.ID, n.Path)
				}
			})
		},
	}
}
