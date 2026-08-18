package cli

import (
	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/pkg/alaws"
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
			books, err := alaws.Discover(flagRoot)
			if err != nil {
				return err
			}
			return printResult(cmd, books, func() {
				for _, b := range books {
					cmd.Printf("%s  %s\n", b.Path, bookLabel(b))
				}
			})
		},
	}
}

func newBooksCreateCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "create [path]",
		Short: "Create a new book at path (default: --root)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := flagRoot
			if len(args) == 1 {
				path = args[0]
			}
			return runBooksCreate(cmd, path, title)
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
	if err := alaws.CreateBook(path, title); err != nil {
		return err
	}
	cmd.Printf("created %s/alaws.toml\n", path)
	return nil
}

// BookInfo is the JSON/human shape of `alaws books show`: the book's own
// title (PLAN1 §4) alongside its ordering tree.
type BookInfo struct {
	Title    string       `json:"title"`
	Sections []alaws.Node `json:"sections"`
}

func newBooksShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [path]",
		Short: "Show a book's title, ordering tree, and metadata",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}
			title, err := alaws.Title(book)
			if err != nil {
				return err
			}
			nodes, err := alaws.Tree(book)
			if err != nil {
				return err
			}
			info := BookInfo{Title: title, Sections: nodes}
			return printResult(cmd, info, func() {
				cmd.Printf("%s  (%s)\n", title, book)
				for _, n := range nodes {
					cmd.Printf("  level %d  %s  (%s)\n", n.Level, n.ID, n.Path)
				}
			})
		},
	}
}
