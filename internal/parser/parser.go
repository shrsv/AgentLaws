// Package parser parses alaws.toml and section Markdown files into raw,
// unvalidated data ready for the compiler. See docs/PLAN1.md §6-§11, §34.
package parser

import (
	"errors"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned by every stub in this package until the
// parser is implemented per PLAN1 §64 Milestone 1.
var ErrNotImplemented = errors.New("parser: not implemented")

// ParsedSection is the raw result of parsing one section file, before
// validation or numbering.
type ParsedSection struct {
	ID         string
	Title      string
	Level      *int // nil if not set in frontmatter
	Commentary string
	RawLaws    []string // one entry per numbered list item found in the laws region
	Source     model.SourceRef
}

// ParseLawbookConfig parses an alaws.toml file.
func ParseLawbookConfig(path string) (model.LawbookMetadata, error) {
	return model.LawbookMetadata{}, ErrNotImplemented
}

// ParseSection parses one Markdown section file into frontmatter,
// commentary, and raw law clauses.
func ParseSection(path string) (ParsedSection, error) {
	return ParsedSection{}, ErrNotImplemented
}
