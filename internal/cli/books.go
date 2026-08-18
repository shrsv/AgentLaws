package cli

import (
	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/discovery"
	"github.com/athreyac4/agentlaws/internal/ordering"
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
					cmd.Println(c.Path)
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

func newBooksShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <path>",
		Short: "Show a book's ordering tree and metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodes, err := ordering.Tree(configPath(args[0]))
			if err != nil {
				return err
			}
			return printResult(cmd, nodes, func() {
				for _, n := range nodes {
					cmd.Printf("level %d  %s  (%s)\n", n.Level, n.ID, n.Path)
				}
			})
		},
	}
}
