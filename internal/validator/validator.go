// Package validator checks a compiled, numbered lawbook for structural
// problems and produces diagnostics. See docs/PLAN1.md §11, §19-§21.
package validator

import (
	"fmt"

	"github.com/athreyac4/agentlaws/internal/model"
	"github.com/athreyac4/agentlaws/internal/template"
)

// Severity distinguishes problems that invalidate a lawbook from problems
// that merely deserve attention (PLAN1 §20).
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warning"
}

// Diagnostic is a single structured validation finding (PLAN1 §19).
//
// Code is one of: missing-config, missing-file, unused-file, missing-title,
// missing-id, duplicate-id, missing-commentary, missing-laws, invalid-laws,
// invalid-ordering, invalid-metadata, invalid-template, ambiguous-numbering.
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Source   *model.SourceRef
}

// HasErrors reports whether diags contains any error-severity diagnostic.
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// CountErrors returns the number of error-severity diagnostics in diags.
func CountErrors(diags []Diagnostic) int {
	n := 0
	for _, d := range diags {
		if d.Severity == SeverityError {
			n++
		}
	}
	return n
}

// Validate checks already-numbered sections for duplicate IDs, ambiguous
// numbering, empty laws regions, and malformed {{template}} placeholders
// (PLAN1 §17a). Checks that depend on the raw ordering/filesystem
// (missing-file, unused-file) are performed by the compiler, which has
// that context.
func Validate(sections []model.Section) []Diagnostic {
	var diags []Diagnostic
	seen := map[string]string{}

	// A section's own laws are numbered <section-number>.<N>, the same
	// scheme used for its child sections' numbers. A section that has both
	// would produce two different things sharing one citation (e.g. a law
	// and a subsection both numbered "1.1"), so the two are mutually
	// exclusive.
	hasChildren := map[string]bool{}
	for _, s := range sections {
		if s.ParentID != "" {
			hasChildren[s.ParentID] = true
		}
	}

	for _, s := range sections {
		if prev, ok := seen[s.ID]; ok {
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     "duplicate-id",
				Message:  fmt.Sprintf("id %q is used by both %s and %s", s.ID, prev, s.Source.Path),
				Source:   &s.Source,
			})
		}
		seen[s.ID] = s.Source.Path

		switch {
		case hasChildren[s.ID] && len(s.Laws) > 0:
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     "ambiguous-numbering",
				Message: fmt.Sprintf(
					"%s: has both child sections and %d law(s) of its own, which produces ambiguous citations; move these laws into a child section",
					s.ID, len(s.Laws)),
				Source: &s.Source,
			})
		case !hasChildren[s.ID] && len(s.Laws) == 0:
			diags = append(diags, Diagnostic{
				Severity: SeverityWarning,
				Code:     "missing-laws",
				Message:  fmt.Sprintf("%s: laws region has no numbered clauses", s.ID),
				Source:   &s.Source,
			})
		}

		if err := template.ValidateSyntax(s.Commentary); err != nil {
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     "invalid-template",
				Message:  fmt.Sprintf("%s: commentary: %v", s.ID, err),
				Source:   &s.Source,
			})
		}
		for _, law := range s.Laws {
			if err := template.ValidateSyntax(law.Text); err != nil {
				src := law.Source
				diags = append(diags, Diagnostic{
					Severity: SeverityError,
					Code:     "invalid-template",
					Message:  fmt.Sprintf("%s: law %s: %v", s.ID, law.Number, err),
					Source:   &src,
				})
			}
		}
	}

	return diags
}
