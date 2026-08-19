package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shrsv/AgentLaws/internal/compiler"
	"github.com/shrsv/AgentLaws/internal/lawedit"
	"github.com/shrsv/AgentLaws/internal/parser"
	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func newLawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "law",
		Short: "Manage individual numbered clauses within a section",
	}
	cmd.AddCommand(newLawAddCmd(), newLawListCmd(), newLawRemoveCmd(), newLawSlugCmd(), newLawFillSlugsCmd())
	return cmd
}

func newLawAddCmd() *cobra.Command {
	var bookFlag string
	var after int
	var slugFlag string
	cmd := &cobra.Command{
		Use:   "add <section-id> <text>",
		Short: "Append a new numbered clause to a section's laws",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID, text := args[0], args[1]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			if flagDryRun {
				path, err := alaws.SectionFilePath(book, sectionID)
				if err != nil {
					return err
				}
				cmd.Printf("would add clause to %s: %q\n", path, text)
				return nil
			}
			return alaws.AddLaw(book, sectionID, text, slugFlag, after)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().IntVar(&after, "after", 0, "insert after this existing clause number")
	cmd.Flags().StringVar(&slugFlag, "slug", "", "explicit slug (auto-generated if empty)")
	return cmd
}

func newLawListCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "list <section-id>",
		Short: "List a section's numbered clauses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID := args[0]
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			laws, err := alaws.ListLaws(book, sectionID)
			if err != nil {
				return err
			}
			return printResult(cmd, laws, func() {
				for i, text := range laws {
					cmd.Printf("%d. %s\n", i+1, text)
				}
			})
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newLawRemoveCmd() *cobra.Command {
	var bookFlag string
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <section-id> <number>",
		Short: "Remove a numbered clause from a section",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID, numStr := args[0], args[1]
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return &UsageError{Msg: fmt.Sprintf("invalid clause number %q", numStr)}
			}
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			if flagDryRun {
				path, err := alaws.SectionFilePath(book, sectionID)
				if err != nil {
					return err
				}
				cmd.Printf("would remove clause %d from %s\n", n, path)
				return nil
			}
			return alaws.RemoveLaw(book, sectionID, n, force)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().BoolVar(&force, "force", false, "remove without confirmation")
	return cmd
}

func newLawSlugCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "slug <section-id> <citation-or-slug> <new-slug>",
		Short: "Change a law's slug",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sectionID, citation, newSlug := args[0], args[1], args[2]
			if !lawedit.IsValidSlug(newSlug) {
				return &UsageError{Msg: fmt.Sprintf("invalid slug %q: must match [a-z][a-z0-9-]*", newSlug)}
			}
			book, err := resolveBook(bookFlag)
			if err != nil {
				return err
			}
			path, err := alaws.SectionFilePath(book, sectionID)
			if err != nil {
				return err
			}
			if flagDryRun {
				cmd.Printf("would set slug of %s in %s to %q\n", citation, path, newSlug)
				return nil
			}
			return lawedit.SetSlug(path, citation, newSlug)
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newLawFillSlugsCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "fill-slugs [book]",
		Short: "Assign slugs to all laws that are missing one",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			book, err := resolveBook(firstArg(args))
			if err != nil {
				return err
			}

			configPath := compiler.ConfigPath(book)
			dir := filepath.Dir(configPath)
			meta, err := parser.ParseLawbookConfig(configPath)
			if err != nil {
				return err
			}

			totalFilled := 0
			for _, entry := range meta.Ordering {
				full := filepath.Join(dir, entry)
				parsed, err := parser.ParseSection(full)
				if err != nil {
					continue
				}

				// Collect existing slugs for de-duplication.
				var existing []string
				needsFill := false
				for _, rl := range parsed.RawLaws {
					if rl.Slug != "" {
						existing = append(existing, rl.Slug)
					} else {
						needsFill = true
					}
				}
				if !needsFill {
					continue
				}

				// Generate slugs for laws missing them.
				for idx, rl := range parsed.RawLaws {
					if rl.Slug != "" {
						continue
					}
					slug := lawedit.GenerateSlug(rl.Text, existing)
					existing = append(existing, slug)
					if flagDryRun {
						cmd.Printf("would fill slug in %s: %q -> %q\n", full, truncate(rl.Text, 40), slug)
						continue
					}
					citation := strconv.Itoa(idx + 1)
					if err := lawedit.SetSlug(full, citation, slug); err != nil {
						cmd.PrintErrf("warning: %s: could not set slug for law %d: %v\n", full, idx+1, err)
						continue
					}
					totalFilled++
				}
			}

			if flagDryRun {
				return nil
			}
			cmd.Printf("filled %d slug(s)\n", totalFilled)
			return nil
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}

func newResolveCmd() *cobra.Command {
	var bookFlag string
	cmd := &cobra.Command{
		Use:   "resolve <citation>",
		Short: "Resolve a canonical citation (e.g. 2.5.3) to its source",
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
					cmd.Printf("%s %s\n  section: %s\n  source:  %s:%d-%d\n",
						r.Law.Number, r.Law.Text, r.Law.SectionID, r.Law.Source.Path, r.Law.Source.LineStart, r.Law.Source.LineEnd)
				case resolver.KindSection:
					cmd.Printf("%s %s (%s)\n  source:  %s:%d-%d\n",
						r.Section.Number, r.Section.Title, r.Section.ID, r.Section.Source.Path, r.Section.Source.LineStart, r.Section.Source.LineEnd)
				}
			})
		},
	}
	cmd.Flags().StringVar(&bookFlag, "book", "", "book path (optional if it can be inferred)")
	return cmd
}
