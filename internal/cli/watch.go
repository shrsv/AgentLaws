package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/internal/server"
	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func newWatchCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "watch [book]",
		Short: "Recompile a book on change and serve the live UI",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}

			events, stop, err := alaws.Watch(book)
			if err != nil {
				return err
			}
			defer stop()

			server.SetRoot(flagRoot)
			go func() {
				addr := fmt.Sprintf(":%d", port)
				cmd.Printf("serving UI on http://localhost%s\n", addr)
				if err := server.ListenAndServe(addr); err != nil {
					cmd.PrintErrln("serve:", err)
				}
			}()

			cmd.Printf("watching %s\n", book)
			for ev := range events {
				for _, d := range ev.Book.Diagnostics() {
					cmd.PrintErrf("%s: %s: %s: %s\n", book, d.Severity, d.Code, d.Message)
				}
				if ev.Err != nil {
					cmd.PrintErrln("compile failed:", ev.Err)
					continue
				}
				outDir := book + "/.alaws/build"
				if err := ev.Book.WriteArtifacts(outDir, "html,json"); err != nil {
					cmd.PrintErrln("write artifacts:", err)
					continue
				}
				cmd.Printf("recompiled %s -> %s\n", book, outDir)
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
		Short: "Serve the UI, optionally pinned to a single book",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, ok := resolveBookForUI(firstArg(args))
			server.SetRoot(flagRoot)
			addr := fmt.Sprintf(":%d", port)
			if ok {
				cmd.Printf("serving %s on http://localhost%s\n", book, addr)
			} else {
				cmd.Printf("serving on http://localhost%s (no single book resolved; pick one in the browser)\n", addr)
			}
			return server.ListenAndServe(addr)
		},
	}
	cmd.Flags().IntVar(&port, "port", 8420, "local UI port")
	return cmd
}
