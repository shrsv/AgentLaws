// Package validator checks a compiled, numbered lawbook for structural
// problems and produces diagnostics. See docs/PLAN1.md §11, §19-§21 and
// docs/linking.md §5.
package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/internal/template"
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
// invalid-ordering, invalid-metadata, invalid-template, ambiguous-numbering,
// unfenced-json, missing-slug, invalid-slug, duplicate-slug,
// ambiguous-identity, dangling-reference, missing-prompt-template,
// empty-prompt-template, dangling-prompt-reference, circular-prompt-reference.
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

// slugRe is the canonical slug charset: lowercase letter, then
// lowercase-alphanumerics and hyphens.
var slugRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// alawsLinkRe matches alaws: link destinations in Markdown.
var alawsLinkRe = regexp.MustCompile(`\(alaws:([^)]+)\)`)

// Validate checks already-numbered sections for duplicate IDs, ambiguous
// numbering, empty laws regions, malformed {{template}} placeholders
// (PLAN1 §17a), and linking diagnostics (docs/linking.md §5). Checks that
// depend on the raw ordering/filesystem (missing-file, unused-file) are
// performed by the compiler, which has that context.
func Validate(book model.Lawbook) []Diagnostic {
	sections := book.Sections
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

	// Collect all section IDs for ambiguous-identity check.
	sectionIDs := map[string]bool{}
	for _, s := range sections {
		sectionIDs[s.ID] = true
	}

	// Check prompt IDs share the section-ID namespace (duplicate-id).
	for _, p := range book.Prompts {
		if prev, ok := seen[p.ID]; ok {
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     "duplicate-id",
				Message:  fmt.Sprintf("id %q is used by both %s and %s", p.ID, prev, p.Source.Path),
				Source:   &p.Source,
			})
		}
		seen[p.ID] = p.Source.Path

		// Validate template syntax in the final Template field.
		if err := template.ValidateSyntax(p.Template); err != nil {
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     "invalid-template",
				Message:  fmt.Sprintf("%s: prompt template: %v", p.ID, err),
				Source:   &p.Source,
			})
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
		if unfencedJSON(s.Commentary) {
			diags = append(diags, Diagnostic{
				Severity: SeverityWarning,
				Code:     "unfenced-json",
				Message: fmt.Sprintf(
					"%s: commentary: JSON written in single backticks will render as inline code, not a highlighted block; use a ```json fenced block instead",
					s.ID),
				Source: &s.Source,
			})
		}

		// Per-section slug uniqueness tracking.
		sectionSlugs := map[string]*model.SourceRef{}

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
			if unfencedJSON(law.Text) {
				src := law.Source
				diags = append(diags, Diagnostic{
					Severity: SeverityWarning,
					Code:     "unfenced-json",
					Message: fmt.Sprintf(
						"%s: law %s: JSON written in single backticks will render as inline code, not a highlighted block; use a ```json fenced block instead",
						s.ID, law.Number),
					Source: &src,
				})
			}

			// missing-slug
			if law.Slug == "" {
				src := law.Source
				diags = append(diags, Diagnostic{
					Severity: SeverityError,
					Code:     "missing-slug",
					Message:  fmt.Sprintf("%s: law %s: has no {#slug}", s.ID, law.Number),
					Source:   &src,
				})
			}

			// invalid-slug
			if law.Slug != "" && !slugRe.MatchString(law.Slug) {
				src := law.Source
				diags = append(diags, Diagnostic{
					Severity: SeverityError,
					Code:     "invalid-slug",
					Message:  fmt.Sprintf("%s: law %s: slug %q does not match [a-z][a-z0-9-]*", s.ID, law.Number, law.Slug),
					Source:   &src,
				})
			}

			// duplicate-slug (within same section)
			if law.Slug != "" {
				if prev, ok := sectionSlugs[law.Slug]; ok {
					src := law.Source
					diags = append(diags, Diagnostic{
						Severity: SeverityError,
						Code:     "duplicate-slug",
						Message:  fmt.Sprintf("%s: law %s: slug %q is already used by another law in this section (at %s:%d)", s.ID, law.Number, law.Slug, prev.Path, prev.LineStart),
						Source:   &src,
					})
				} else {
					src := law.Source
					sectionSlugs[law.Slug] = &src
				}
			}

			// ambiguous-identity: law's FQ identity matches another section's ID
			if law.Slug != "" {
				fqID := s.ID + "." + law.Slug
				if sectionIDs[fqID] {
					src := law.Source
					diags = append(diags, Diagnostic{
						Severity: SeverityWarning,
						Code:     "ambiguous-identity",
						Message:  fmt.Sprintf("%s: law %s: fully-qualified identity %q shadows section ID %q", s.ID, law.Number, fqID, fqID),
						Source:   &src,
					})
				}
			}
		}

		// dangling-reference: scan commentary and law text for alaws: links
		diags = append(diags, checkDanglingRefs(book, s.ID, s.Commentary, &s.Source)...)
		for _, law := range s.Laws {
			diags = append(diags, checkDanglingRefs(book, s.ID, law.Text, &law.Source)...)
		}
	}

	return diags
}

// checkDanglingRefs scans text for alaws: link destinations and reports
// any that fail to resolve.
func checkDanglingRefs(book model.Lawbook, sectionID, text string, src *model.SourceRef) []Diagnostic {
	var diags []Diagnostic
	for _, m := range alawsLinkRe.FindAllStringSubmatch(text, -1) {
		token := m[1]
		if _, err := resolver.Resolve(book, token); err != nil {
			diags = append(diags, Diagnostic{
				Severity: SeverityWarning,
				Code:     "dangling-reference",
				Message:  fmt.Sprintf("%s: alaws:%s does not resolve: %v", sectionID, token, err),
				Source:   src,
			})
		}
	}
	return diags
}

// inlineCodeSpan matches a single- or double-backtick inline code span.
// [^`] is a negated class, so it already matches newlines - a span the
// author wrapped across lines (one backtick around a whole block) is
// exactly what we want to catch, and Go's regexp needs no (?s) flag for it.
var inlineCodeSpan = regexp.MustCompile("`([^`]+)`")

// unfencedJSON reports whether text contains JSON that an author meant as a
// fenced block but wrote with single backticks instead of ```json. Such a
// span renders as inline code, not a highlighted block. It matches a
// backtick span whose content is a json info string plus more content, a
// span that begins with a JSON object/array literal, or a lone "`json" line
// (an unclosed single-backtick opener). Proper ```json fenced blocks are
// stripped out first, so they are never flagged.
func unfencedJSON(text string) bool {
	text = stripFencedBlocks(text)
	for _, m := range inlineCodeSpan.FindAllStringSubmatch(text, -1) {
		content := strings.TrimSpace(m[1])
		lower := strings.ToLower(content)
		if strings.HasPrefix(lower, "json") && len(content) > len("json") {
			return true
		}
		if jsonish(content) {
			return true
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "`json") {
			rest := strings.TrimSpace(trimmed[len("`json"):])
			if rest == "" || strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[") {
				return true
			}
		}
	}
	return false
}

// jsonish reports whether an inline code span's content looks like a JSON
// object or array: it begins with { or [, and its first colon follows a
// quoted or bare key. Constructs that merely resemble JSON are excluded -
// a Markdown link reference (`[foo]: url`, colon after ]) and a brace
// label (`{a}: {b}`, colon after }) - so the heuristic stays narrow.
func jsonish(content string) bool {
	if !strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
		return false
	}
	i := strings.Index(content, ":")
	if i <= 0 {
		return false
	}
	switch content[i-1] {
	case '"', '\'':
		return true
	case ']':
		return false
	}
	return isWordByte(content[i-1])
}

func isWordByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

// stripFencedBlocks removes ``` and ~~~ fenced regions so inline-span
// detection below cannot mistake the backticks of a proper code block for
// single-tick delimiters.
func stripFencedBlocks(text string) string {
	inFence := false
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
