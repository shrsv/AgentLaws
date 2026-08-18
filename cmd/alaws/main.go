// Command alaws is the AgentLaws command-line interface. See
// docs/PLAN1.md §32 for the full command reference.
package main

import (
	"fmt"
	"os"

	"github.com/shrsv/AgentLaws/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "alaws:", err)
		os.Exit(cli.ExitCode(err))
	}
}
