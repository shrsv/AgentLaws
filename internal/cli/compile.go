package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/provenance"
	"github.com/athreyac4/agentlaws/internal/resolver"
)

func newCompileCmd() *cobra.Command {
	var out, format string
	var strict bool
	cmd := &cobra.Command{
		Use:   "compile [book...]",
		Short: "Compile one or more books into a deterministic Lawbook IR and artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			books := args
			if len(books) == 0 {
				books = []string{flagRoot}
			}
			for _, book := range books {
				result, err := compiler.Compile(book, compiler.Options{Strict: strict})
				if err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}
				cmd.Printf("compiled %s: %d sections, %d diagnostics\n", book, len(result.Lawbook.Sections), len(result.Diagnostics))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory for compiled artifacts")
	cmd.Flags().StringVar(&format, "format", "html,json", "comma-separated artifact formats: html,json,pdf")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as errors")
	return cmd
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [book...]",
		Short: "Validate one or more books without producing artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			books := args
			if len(books) == 0 {
				books = []string{flagRoot}
			}
			for _, book := range books {
				result, err := compiler.Compile(book, compiler.Options{})
				if err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}
				if err := printResult(cmd, result.Diagnostics, func() {
					for _, d := range result.Diagnostics {
						cmd.Printf("%s: %s: %s\n", book, d.Code, d.Message)
					}
				}); err != nil {
					return err
				}
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
			book := flagRoot
			if len(args) == 1 {
				book = args[0]
			}
			result, err := compiler.Compile(book, compiler.Options{})
			if err != nil {
				return err
			}
			return printResult(cmd, result.Lawbook.Sections, func() {
				for _, s := range result.Lawbook.Sections {
					cmd.Printf("%s %s (%s)\n", s.Number, s.Title, s.ID)
					for _, law := range s.Laws {
						cmd.Printf("  %s %s\n", law.Number, law.Text)
					}
				}
			})
		},
	}
}

func loadBook(book string) (compiler.Result, error) {
	return compiler.Compile(book, compiler.Options{})
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <citation-or-id>",
		Short: "Show a law or section by citation or stable ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := loadBook(flagRoot)
			if err != nil {
				return err
			}
			if law, err := resolver.ResolveLaw(result.Lawbook, args[0]); err == nil {
				return printResult(cmd, law, func() {
					cmd.Printf("%s %s\n", law.Number, law.Text)
				})
			}
			section, err := resolver.ResolveSection(result.Lawbook, args[0])
			if err != nil {
				return err
			}
			return printResult(cmd, section, func() {
				cmd.Printf("%s %s (%s)\n", section.Number, section.Title, section.ID)
			})
		},
	}
}

func newResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <citation>",
		Short: "Resolve a canonical citation (e.g. 2.5.3) to its source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := loadBook(flagRoot)
			if err != nil {
				return err
			}
			law, err := resolver.ResolveLaw(result.Lawbook, args[0])
			if err != nil {
				return err
			}
			return printResult(cmd, law, func() {
				cmd.Printf("%s %s\n  section: %s\n  source:  %s:%d-%d\n",
					law.Number, law.Text, law.SectionID, law.Source.Path, law.Source.LineStart, law.Source.LineEnd)
			})
		},
	}
}

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history <citation>",
		Short: "Show the Git history of a law",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := loadBook(flagRoot)
			if err != nil {
				return err
			}
			hist, err := provenance.History(result.Lawbook, args[0])
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
}
