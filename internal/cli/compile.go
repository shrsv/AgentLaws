package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func newCompileCmd() *cobra.Command {
	var out, format string
	cmd := &cobra.Command{
		Use:   "compile [book...]",
		Short: "Compile one or more books into a deterministic Lawbook IR and artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			books, err := resolveBooks(args)
			if err != nil {
				return err
			}
			for _, book := range books {
				b, err := alaws.Compile(book)
				for _, d := range b.Diagnostics() {
					cmd.PrintErrf("%s: %s: %s\n", book, d.Code, d.Message)
				}
				if err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}

				outDir := out
				if outDir == "" {
					outDir = book + "/.alaws/build"
				}
				if flagDryRun {
					cmd.Printf("would write %s to %s (%s)\n", book, outDir, format)
					continue
				}
				if err := b.WriteArtifacts(outDir, format); err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}
				cmd.Printf("compiled %s: %d sections, %d diagnostics -> %s\n",
					book, len(b.Lawbook().Sections), len(b.Diagnostics()), outDir)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory for compiled artifacts (default <book>/.alaws/build)")
	cmd.Flags().StringVar(&format, "format", "html,json", "comma-separated artifact formats: html,json,pdf,md")
	return cmd
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [book...]",
		Short: "Validate one or more books without producing artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			books, err := resolveBooks(args)
			if err != nil {
				return err
			}
			var failed []string
			for _, book := range books {
				// alaws.Compile returns an error both when the lawbook
				// can't be read at all (Diagnostics is then empty) and
				// when it was read but contains error-severity
				// diagnostics; either way, validate's whole job is to
				// show what it found, so it must print before deciding
				// whether to fail.
				b, err := alaws.Compile(book)
				if perr := printResult(cmd, b.Diagnostics(), func() {
					if len(b.Diagnostics()) == 0 {
						cmd.Printf("%s: OK\n", book)
						return
					}
					for _, d := range b.Diagnostics() {
						cmd.Printf("%s: %s: %s: %s\n", book, d.Severity, d.Code, d.Message)
					}
				}); perr != nil {
					return perr
				}
				if err != nil {
					cmd.PrintErrf("%s: %v\n", book, err)
					failed = append(failed, book)
				}
			}
			if len(failed) > 0 {
				return fmt.Errorf("validation failed for: %s", strings.Join(failed, ", "))
			}
			return nil
		},
	}
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [book]",
		Short: "List compiled sections and laws with canonical numbers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			return printResult(cmd, b.Lawbook(), func() {
				cmd.Printf("%s\n", b.Lawbook().Metadata.Title)
				for _, s := range b.Lawbook().Sections {
					cmd.Printf("%s %s (%s)\n", s.Number, s.Title, s.ID)
					for _, law := range s.Laws {
						cmd.Printf("  %s %s\n", law.Number, law.Text)
					}
				}
			})
		},
	}
}

func newShowCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "show <citation-or-id>",
		Short: "Show a law or section by citation, slug, or stable ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			r, err := resolver.Resolve(b.Lawbook(), args[0])
			if err != nil {
				return err
			}
			return printResult(cmd, r, func() {
				switch r.Kind {
				case resolver.KindLaw:
					cmd.Printf("%s %s\n", r.Law.Number, r.Law.Text)
				case resolver.KindSection:
					cmd.Printf("%s %s (%s)\n", r.Section.Number, r.Section.Title, r.Section.ID)
				}
			})
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newHistoryCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "history <citation>",
		Short: "Show the Git history of a law",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			b, err := alaws.Compile(book)
			if err != nil {
				return err
			}
			hist, err := b.History(args[0])
			if err != nil {
				return err
			}
			return printResult(cmd, hist, func() {
				cmd.Printf("%s introduced in %s\n", hist.Citation, hist.Introduced)
				for _, m := range hist.Modifications {
					cmd.Printf("  %s  %s  %s\n", m.Commit, m.Author, m.Summary)
				}
			})
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}
