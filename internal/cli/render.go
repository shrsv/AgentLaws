package cli

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

func newRenderCmd() *cobra.Command {
	var book, section, law string
	var all bool
	var vars []string
	var varsFile string
	var onMissing string

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render selected laws as prompt-ready text, substituting {{variables}}",
		Long: `render extracts laws from a compiled book and substitutes {{variable}}
placeholders, producing text ready to embed in an application's prompt
(see docs/PLAN1.md §17a). Select laws with exactly one of --section,
--law, or --all.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if section == "" && law == "" && !all {
				return &UsageError{Msg: "one of --section, --law, or --all is required"}
			}

			resolved, err := resolveBook(book)
			if err != nil {
				return err
			}
			b, err := alaws.Load(resolved)
			if err != nil {
				return err
			}

			sel := alaws.Selector{All: all}
			if section != "" {
				sel.SectionIDs = []string{section}
			}
			if law != "" {
				sel.Citations = []string{law}
			}
			laws, err := b.Laws(sel)
			if err != nil {
				return err
			}

			mergedVars, err := mergeVars(vars, varsFile)
			if err != nil {
				return err
			}

			policy, err := parseMissingPolicy(onMissing)
			if err != nil {
				return err
			}

			rendered, err := laws.Render(alaws.RenderOptions{Vars: mergedVars, OnMissing: policy})
			if err != nil {
				return err
			}
			return printResult(cmd, rendered, func() {
				cmd.Println(rendered)
			})
		},
	}

	cmd.Flags().StringVar(&book, "book", "", "book path (optional if it can be inferred)")
	cmd.Flags().StringVar(&section, "section", "", "render all laws in this section ID")
	cmd.Flags().StringVar(&law, "law", "", "render a single law by citation")
	cmd.Flags().BoolVar(&all, "all", false, "render every law in the book")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable in key=value form (repeatable)")
	cmd.Flags().StringVar(&varsFile, "vars-file", "", "path to a JSON or YAML file of variables")
	cmd.Flags().StringVar(&onMissing, "on-missing", "error", "error|keep|empty")

	return cmd
}

func parseMissingPolicy(s string) (alaws.MissingPolicy, error) {
	switch s {
	case "", "error":
		return alaws.MissingError, nil
	case "keep":
		return alaws.MissingKeepPlaceholder, nil
	case "empty":
		return alaws.MissingEmpty, nil
	default:
		return 0, &UsageError{Msg: "--on-missing must be one of error|keep|empty, got " + s}
	}
}

// mergeVars combines --var flags (highest precedence) with a --vars-file
// (JSON or YAML flat string map), per the precedence documented in
// docs/PLAN1.md §17a.
func mergeVars(varFlags []string, varsFile string) (map[string]string, error) {
	result := map[string]string{}

	if varsFile != "" {
		data, err := os.ReadFile(varsFile)
		if err != nil {
			return nil, err
		}
		if err := decodeVarsFile(varsFile, data, result); err != nil {
			return nil, err
		}
	}

	for _, kv := range varFlags {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, &UsageError{Msg: "--var must be in key=value form, got " + kv}
		}
		result[k] = v
	}

	return result, nil
}

func decodeVarsFile(path string, data []byte, into map[string]string) error {
	if strings.HasSuffix(path, ".json") {
		return json.Unmarshal(data, &into)
	}
	return decodeYAMLVars(data, into)
}
