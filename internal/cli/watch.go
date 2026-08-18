package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/server"
	"github.com/athreyac4/agentlaws/internal/watcher"
)

func newWatchCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "watch [book]",
		Short: "Recompile a book on change and serve the live UI",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book := flagRoot
			if len(args) == 1 {
				book = args[0]
			}
			events, stop, err := watcher.Watch(book)
			if err != nil {
				return err
			}
			defer stop()
			cmd.Printf("watching %s, serving on :%d\n", book, port)
			for ev := range events {
				if ev.Err != nil {
					cmd.PrintErrln("compile error:", ev.Err)
					continue
				}
				cmd.Println("recompiled", ev.ClusterPath)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&port, "port", 8420, "local UI port")
	return cmd
}

func newServeCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "serve [book]",
		Short: "Serve the UI read-only, without a filesystem watcher",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// The book argument will select which lawbook the UI's Lawbook
			// API serves once internal/server exposes one (PLAN1 §64
			// Milestone 9); today only the static UI shell is served.
			addr := fmt.Sprintf(":%d", port)
			cmd.Printf("serving on http://localhost%s\n", addr)
			return server.ListenAndServe(addr)
		},
	}
	cmd.Flags().IntVar(&port, "port", 8420, "local UI port")
	return cmd
}
