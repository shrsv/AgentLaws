package cli

import (
	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/pkg/alaws"
)

// logChange is one commit's metadata plus what it changed in the lawbook -
// the JSON/human shape shared by `alaws log` and `alaws diff`.
type logChange struct {
	Commit  string            `json:"commit,omitempty"`
	Author  string            `json:"author,omitempty"`
	Date    string            `json:"date,omitempty"`
	Summary string            `json:"summary,omitempty"`
	Diff    alaws.LawbookDiff `json:"diff"`
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func printDiff(cmd *cobra.Command, diff alaws.LawbookDiff) {
	for _, m := range diff.ModifiedLaws {
		cmd.Printf("  ~ %s -> %s: %q -> %q\n", m.OldNumber, m.NewNumber, truncate(m.OldText, 60), truncate(m.NewText, 60))
	}
	for _, l := range diff.AddedLaws {
		cmd.Printf("  + %s: %q\n", l.Number, truncate(l.Text, 60))
	}
	for _, l := range diff.RemovedLaws {
		cmd.Printf("  - %s: %q\n", l.Number, truncate(l.Text, 60))
	}
	for _, id := range diff.AddedSections {
		cmd.Printf("  + section %s\n", id)
	}
	for _, id := range diff.RemovedSections {
		cmd.Printf("  - section %s\n", id)
	}
}

func newLogCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "log [book]",
		Short: "Show the chronological change history of a lawbook (docs/PLAN1.md §37-39)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}

			entries, err := alaws.Log(book, limit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				return printResult(cmd, []logChange{}, func() { cmd.Println("no history") })
			}

			compiled := make(map[string]*alaws.Book, len(entries))
			for _, e := range entries {
				b, err := alaws.CompileRevision(book, e.Commit)
				if err != nil {
					return err
				}
				compiled[e.Commit] = b
			}

			changes := make([]logChange, len(entries))
			for i, e := range entries {
				c := logChange{Commit: e.Commit, Author: e.Author, Date: e.Date, Summary: e.Summary}
				if i+1 < len(entries) {
					c.Diff = alaws.Diff(compiled[entries[i+1].Commit], compiled[e.Commit])
				}
				changes[i] = c
			}

			return printResult(cmd, changes, func() {
				for _, c := range changes {
					cmd.Printf("%s  %s  %s\n", shortHash(c.Commit), c.Date, c.Author)
					cmd.Printf("    %s\n", c.Summary)
					printDiff(cmd, c.Diff)
				}
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum number of commits to show")
	return cmd
}

func newDiffCmd() *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "diff <book>",
		Short: "Show what changed in a lawbook between two Git revisions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return &UsageError{Msg: "--from is required (e.g. --from HEAD~5)"}
			}
			book, err := resolveBook(args[0])
			if err != nil {
				return err
			}

			oldBook, err := alaws.CompileRevision(book, from)
			if err != nil {
				return err
			}
			newBook, err := alaws.CompileRevision(book, to)
			if err != nil {
				return err
			}
			diff := alaws.Diff(oldBook, newBook)

			return printResult(cmd, logChange{Diff: diff}, func() {
				cmd.Printf("%s..%s\n", from, to)
				printDiff(cmd, diff)
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "revision to diff from (required)")
	cmd.Flags().StringVar(&to, "to", "HEAD", "revision to diff to")
	return cmd
}
