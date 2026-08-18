// Package compiler drives the compilation pipeline described in
// docs/PLAN1.md §18: load config -> parse sections -> assign numbering ->
// run diagnostics -> construct the Lawbook IR.
package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/athreyac4/agentlaws/internal/discovery"
	"github.com/athreyac4/agentlaws/internal/model"
	"github.com/athreyac4/agentlaws/internal/numbering"
	"github.com/athreyac4/agentlaws/internal/parser"
	"github.com/athreyac4/agentlaws/internal/validator"
)

// Options configures a single Compile call.
type Options struct {
	// Strict causes any warning-level diagnostic to be treated as an error.
	Strict bool
}

// Result is the outcome of compiling one lawbook cluster.
type Result struct {
	Lawbook     model.Lawbook
	Diagnostics []validator.Diagnostic
}

// ConfigPath resolves a book argument (a directory, or an explicit
// alaws.toml path) to the path of its alaws.toml.
func ConfigPath(book string) string {
	if filepath.Base(book) == "alaws.toml" {
		return book
	}
	return filepath.Join(book, "alaws.toml")
}

// Compile compiles the lawbook cluster rooted at path. It returns a
// non-nil error only when the lawbook cannot be deterministically
// understood at all (missing/malformed alaws.toml) or when any
// error-severity diagnostic was found (or, with Strict, any diagnostic at
// all) - per docs/PLAN1.md §20, "if the compiler cannot deterministically
// understand the lawbook, compilation fails." Result is always populated,
// so callers (e.g. `alaws validate`) can inspect Diagnostics even on
// failure.
func Compile(path string, opts Options) (Result, error) {
	configPath := ConfigPath(path)
	dir := filepath.Dir(configPath)

	meta, err := parser.ParseLawbookConfig(configPath)
	if err != nil {
		return Result{}, fmt.Errorf("missing-config: %s: %w", configPath, err)
	}

	var diags []validator.Diagnostic
	var sections []model.Section

	for _, entry := range meta.Ordering {
		full := filepath.Join(dir, entry)
		if _, statErr := os.Stat(full); os.IsNotExist(statErr) {
			diags = append(diags, validator.Diagnostic{
				Severity: validator.SeverityError,
				Code:     "missing-file",
				Message:  fmt.Sprintf("%s is listed in ordering but does not exist", entry),
			})
			continue
		}

		ps, err := parser.ParseSection(full)
		if err != nil {
			diags = append(diags, validator.Diagnostic{
				Severity: validator.SeverityError,
				Code:     "invalid-metadata",
				Message:  fmt.Sprintf("%s: %v", entry, err),
			})
			continue
		}

		sec := model.Section{
			ID:         ps.ID,
			Title:      ps.Title,
			Level:      parser.ResolveLevel(entry, ps.Level),
			Source:     ps.Source,
			Commentary: ps.Commentary,
		}
		for _, rl := range ps.RawLaws {
			sec.Laws = append(sec.Laws, model.Law{
				Text:   rl.Text,
				Source: model.SourceRef{Path: full, LineStart: rl.LineStart, LineEnd: rl.LineEnd},
			})
		}
		sections = append(sections, sec)
	}

	numbered, err := numbering.Assign(sections)
	if err != nil {
		return Result{}, err
	}

	diags = append(diags, validator.Validate(numbered)...)

	if unordered, uErr := discovery.UnorderedFiles(discovery.Cluster{Path: dir, ConfigPath: configPath}, meta.Ordering); uErr == nil {
		for _, f := range unordered {
			diags = append(diags, validator.Diagnostic{
				Severity: validator.SeverityWarning,
				Code:     "unused-file",
				Message:  fmt.Sprintf("%s exists but is not present in ordering", f),
			})
		}
	}

	result := Result{
		Lawbook: model.Lawbook{
			Metadata: meta,
			Sections: numbered,
		},
		Diagnostics: diags,
	}

	if validator.HasErrors(diags) {
		return result, fmt.Errorf("%d error(s) found", validator.CountErrors(diags))
	}
	if opts.Strict && len(diags) > 0 {
		return result, fmt.Errorf("%d diagnostic(s) found (--strict)", len(diags))
	}

	return result, nil
}
