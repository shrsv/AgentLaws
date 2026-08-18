package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/parser"
)

// configPath resolves a book argument (a directory, or an explicit
// alaws.toml path) to the path of its alaws.toml.
func configPath(book string) string {
	return compiler.ConfigPath(book)
}

// levelOverride decides whether a newly created section needs an explicit
// `level:` written into its frontmatter. Level normally defaults from a
// file's folder depth (parser.ResolveLevel); an explicit value is only
// needed - and only written - when desired diverges from what depth alone
// would produce, i.e. the file lives somewhere its intended nesting
// doesn't naturally match. Returns 0 (meaning "omit it") when they agree.
func levelOverride(entryPath string, desired int) int {
	if parser.ResolveLevel(entryPath, nil) == desired {
		return 0
	}
	return desired
}

// printResult prints v as JSON when --json is set, otherwise runs human,
// which is expected to write human-readable output via cmd.Print*.
func printResult(cmd *cobra.Command, v any, human func()) error {
	if flagJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	human()
	return nil
}
