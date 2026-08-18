package cli

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/internal/server"
)

func newUICmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "ui [book]",
		Short: "Open the AgentLaws web UI in a browser",
		Long: `ui starts the local server (the same one 'alaws serve' runs) and opens
it in your default browser. If a book can be resolved - given explicitly,
or because exactly one lives under --root - the browser opens straight to
it; otherwise it opens to the book picker, matching how 'alaws serve'
hands off ambiguity to the UI instead of prompting on stdin (docs/PLAN1.md
§32).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, ok := resolveBookForUI(firstArg(args))
			server.SetRoot(flagRoot)
			target := fmt.Sprintf("http://localhost:%d/#/books", port)
			if ok {
				target = fmt.Sprintf("http://localhost:%d/#/books/%s", port, encodeRouteSegment(book))
				cmd.Printf("opening %s\n", target)
			} else {
				cmd.Printf("opening %s (pick a book in the browser)\n", target)
			}

			go func() {
				// Give the server a brief moment to bind so the browser
				// doesn't hit connection-refused on the very first request.
				time.Sleep(200 * time.Millisecond)
				_ = openBrowser(target)
			}()

			return server.ListenAndServe(fmt.Sprintf(":%d", port))
		},
	}
	cmd.Flags().IntVar(&port, "port", 8420, "local UI port")
	return cmd
}

// encodeRouteSegment matches the frontend router's encodeURIComponent
// (web/src/router.ts) closely enough for filesystem paths: escape
// everything QueryEscape would, but with %20 instead of + for spaces.
func encodeRouteSegment(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}
