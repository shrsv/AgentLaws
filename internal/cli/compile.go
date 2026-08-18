package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/model"
	"github.com/athreyac4/agentlaws/internal/provenance"
	renderhtml "github.com/athreyac4/agentlaws/internal/renderer/html"
	renderpdf "github.com/athreyac4/agentlaws/internal/renderer/pdf"
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
				for _, d := range result.Diagnostics {
					cmd.PrintErrf("%s: %s: %s\n", book, d.Code, d.Message)
				}
				if err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}

				outDir := out
				if outDir == "" {
					outDir = filepath.Join(book, ".alaws", "build")
				}
				if flagDryRun {
					cmd.Printf("would write %s to %s (%s)\n", book, outDir, format)
					continue
				}
				if err := writeArtifacts(outDir, format, result.Lawbook); err != nil {
					return fmt.Errorf("%s: %w", book, err)
				}
				cmd.Printf("compiled %s: %d sections, %d diagnostics -> %s\n", book, len(result.Lawbook.Sections), len(result.Diagnostics), outDir)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory for compiled artifacts (default <book>/.alaws/build)")
	cmd.Flags().StringVar(&format, "format", "html,json", "comma-separated artifact formats: html,json,pdf")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as errors")
	return cmd
}

// writeArtifacts renders book into outDir in each of the comma-separated
// formats, per docs/PLAN1.md §22-§23, §26: every format is a renderer over
// the same Lawbook IR, not a separate parse of the source.
func writeArtifacts(outDir, format string, book model.Lawbook) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	for _, f := range strings.Split(format, ",") {
		switch strings.TrimSpace(f) {
		case "html":
			if err := writeArtifact(filepath.Join(outDir, "lawbook.html"), func(w *os.File) error {
				return renderhtml.Render(w, book)
			}); err != nil {
				return err
			}
		case "pdf":
			if err := writeArtifact(filepath.Join(outDir, "lawbook.pdf"), func(w *os.File) error {
				return renderpdf.Render(w, book)
			}); err != nil {
				return err
			}
		case "json":
			if err := writeArtifact(filepath.Join(outDir, "lawbook.json"), func(w *os.File) error {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(book)
			}); err != nil {
				return err
			}
		case "":
			// allow trailing commas
		default:
			return &UsageError{Msg: "unknown --format value " + f}
		}
	}
	return nil
}

func writeArtifact(path string, render func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return render(f)
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
			var failed []string
			for _, book := range books {
				// Compile() returns an error both when the lawbook can't be
				// read at all (Diagnostics is then empty) and when it was
				// read but contains error-severity diagnostics; either way,
				// validate's whole job is to show what it found, so it must
				// print before deciding whether to fail.
				result, err := compiler.Compile(book, compiler.Options{})
				if perr := printResult(cmd, result.Diagnostics, func() {
					if len(result.Diagnostics) == 0 {
						cmd.Printf("%s: OK\n", book)
						return
					}
					for _, d := range result.Diagnostics {
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
			book := flagRoot
			if len(args) == 1 {
				book = args[0]
			}
			result, err := compiler.Compile(book, compiler.Options{})
			if err != nil {
				return err
			}
			return printResult(cmd, result.Lawbook, func() {
				cmd.Printf("%s\n", result.Lawbook.Metadata.Title)
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
