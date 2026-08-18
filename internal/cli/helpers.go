package cli

import (
	"encoding/json"
	"path/filepath"

	"github.com/spf13/cobra"
)

// configPath resolves a book argument (a directory, or an explicit
// alaws.toml path) to the path of its alaws.toml.
func configPath(book string) string {
	if filepath.Base(book) == "alaws.toml" {
		return book
	}
	return filepath.Join(book, "alaws.toml")
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
