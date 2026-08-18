package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

func newExportCmd() *cobra.Command {
	var out, format, title string
	cmd := &cobra.Command{
		Use:   "export [root]",
		Short: "Export every book under a root as one combined HTML/PDF document",
		Long: `export compiles every book discovered under root and renders them into a
single combined document - unlike 'alaws compile', which produces one set
of artifacts per book. Use this when you want the whole governance
program (every book, not just one) as one file to read or hand out.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := flagRoot
			if len(args) == 1 {
				root = args[0]
			}
			books, err := alaws.CompileAll(root)
			if err != nil {
				// Some books may have failed to compile; report that but
				// still export whatever did compile, same spirit as
				// `compile`'s per-book diagnostics.
				cmd.PrintErrln(err)
			}
			if len(books) == 0 {
				return &UsageError{Msg: fmt.Sprintf("no lawbook found under %q", root)}
			}

			docTitle := title
			if docTitle == "" {
				docTitle = "Combined Lawbook"
			}
			outDir := out
			if outDir == "" {
				outDir = root + "/.alaws/export"
			}
			if flagDryRun {
				cmd.Printf("would write combined export of %d book(s) to %s (%s)\n", len(books), outDir, format)
				return nil
			}
			if err := alaws.WriteCombinedArtifacts(books, docTitle, outDir, format); err != nil {
				return err
			}
			cmd.Printf("exported %d book(s) -> %s\n", len(books), outDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default <root>/.alaws/export)")
	cmd.Flags().StringVar(&format, "format", "html,pdf", "comma-separated formats: html,pdf,md")
	cmd.Flags().StringVar(&title, "title", "", `title for the combined document (default "Combined Lawbook")`)
	return cmd
}
