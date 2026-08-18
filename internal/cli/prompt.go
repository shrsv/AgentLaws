package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// isInteractive reports whether stdin is a terminal a human can respond to
// - false for scripts, pipes, and CI, where prompting would just hang or
// silently consume the wrong input.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// promptChoice prints a numbered list of options (as label strings, one
// per line) and reads a 1-based selection from stdin. Used only after
// isInteractive() has confirmed stdin is a real terminal.
func promptChoice(question string, labels []string) (int, error) {
	fmt.Fprintln(os.Stderr, question)
	for i, l := range labels {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, l)
	}
	fmt.Fprint(os.Stderr, "Select: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	line = strings.TrimSpace(line)
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(labels) {
		return 0, fmt.Errorf("invalid selection %q: enter a number from 1 to %d", line, len(labels))
	}
	return n - 1, nil
}
