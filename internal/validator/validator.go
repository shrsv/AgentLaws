// Package validator checks a parsed lawbook for structural problems and
// produces diagnostics. See docs/PLAN1.md §11, §19-§21, §34.
package validator

import (
	"errors"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned by every stub in this package until
// validation is implemented per PLAN1 §64 Milestone 2.
var ErrNotImplemented = errors.New("validator: not implemented")

// Severity distinguishes problems that invalidate a lawbook from problems
// that merely deserve attention (PLAN1 §20).
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

// Diagnostic is a single structured validation finding (PLAN1 §19).
//
// Code is one of: missing-config, missing-file, unused-file, missing-title,
// missing-id, duplicate-id, missing-commentary, missing-laws, invalid-laws,
// invalid-ordering, invalid-metadata, invalid-template.
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Source   *model.SourceRef
}

// Validate checks a lawbook's config and parsed sections and returns all
// diagnostics found. It does not stop at the first error.
func Validate(meta model.LawbookMetadata, sections []model.Section) ([]Diagnostic, error) {
	return nil, ErrNotImplemented
}
