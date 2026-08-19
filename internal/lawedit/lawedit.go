// Package lawedit locates the `<!-- alaws:laws -->` region in a section
// file and edits its numbered list without disturbing surrounding Markdown.
// It backs `alaws law add`/`alaws law remove` (PLAN1 §32) and the slug
// management commands (docs/linking.md §6).
package lawedit

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/shrsv/AgentLaws/internal/parser"
)

const lawsMarker = "<!-- alaws:laws -->"

// clause is a single law's text and optional slug, as stored in the
// source file's laws region.
type clause struct {
	Text string
	Slug string
}

// splitAtLawsMarker returns everything up to and including the laws
// marker line (unchanged verbatim) and the existing clauses found after
// it. Parsing is fence-aware via parser.ParseLawLines.
func splitAtLawsMarker(content string) (header string, clauses []clause, err error) {
	lines := strings.Split(content, "\n")
	markerIdx := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == lawsMarker {
			markerIdx = i
			break
		}
	}
	if markerIdx == -1 {
		return "", nil, fmt.Errorf("lawedit: %s marker not found", lawsMarker)
	}

	header = strings.Join(lines[:markerIdx+1], "\n")

	rawLaws := parser.ParseLawLines(lines[markerIdx+1:], markerIdx+1)
	for _, rl := range rawLaws {
		clauses = append(clauses, clause{Text: rl.Text, Slug: rl.Slug})
	}
	return header, clauses, nil
}

func writeClauses(path, header string, clauses []clause) error {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i, c := range clauses {
		b.WriteString("\n")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		if strings.Contains(c.Text, "\n") {
			// Multi-line: text on first line, slug on its own trailing line.
			b.WriteString(c.Text)
			if c.Slug != "" {
				b.WriteString("\n{#")
				b.WriteString(c.Slug)
				b.WriteString("}")
			}
		} else {
			// One-line: inline slug.
			b.WriteString(c.Text)
			if c.Slug != "" {
				b.WriteString(" {#")
				b.WriteString(c.Slug)
				b.WriteString("}")
			}
		}
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// Add appends a new numbered clause to the laws region of the section file
// at path. If after > 0, the clause is inserted immediately after that
// existing clause number instead of at the end. If slug is empty, one is
// auto-generated from the text.
func Add(path string, text string, slug string, after int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header, clauses, err := splitAtLawsMarker(string(data))
	if err != nil {
		return err
	}

	if slug == "" {
		slug = GenerateSlug(text, existingSlugs(clauses))
	}

	idx := len(clauses)
	if after > 0 && after <= len(clauses) {
		idx = after
	}
	clauses = append(clauses[:idx:idx], append([]clause{{Text: text, Slug: slug}}, clauses[idx:]...)...)

	return writeClauses(path, header, clauses)
}

// Remove deletes the numbered clause `number` from the laws region of the
// section file at path, renumbering subsequent clauses.
func Remove(path string, number int, force bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header, clauses, err := splitAtLawsMarker(string(data))
	if err != nil {
		return err
	}
	if number < 1 || number > len(clauses) {
		return fmt.Errorf("lawedit: clause %d does not exist (section has %d)", number, len(clauses))
	}

	clauses = append(clauses[:number-1], clauses[number:]...)
	return writeClauses(path, header, clauses)
}

// SetSlug changes the slug of the clause identified by citation (a number
// like "1" or an existing slug) within the section file at path.
func SetSlug(path string, citation string, newSlug string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header, clauses, err := splitAtLawsMarker(string(data))
	if err != nil {
		return err
	}

	// Find the target clause by number or existing slug.
	targetIdx := -1
	for i, c := range clauses {
		num := strconv.Itoa(i + 1)
		if c.Slug == citation || num == citation {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return fmt.Errorf("lawedit: clause %q not found", citation)
	}

	clauses[targetIdx].Slug = newSlug
	return writeClauses(path, header, clauses)
}

// existingSlugs collects all non-empty slugs from clauses.
func existingSlugs(clauses []clause) []string {
	var slugs []string
	for _, c := range clauses {
		if c.Slug != "" {
			slugs = append(slugs, c.Slug)
		}
	}
	return slugs
}

// slugWordRe matches sequences of lowercase letters and digits.
var slugWordRe = regexp.MustCompile(`[a-z][a-z0-9]*`)

// GenerateSlug derives a slug from text: lowercase, strip punctuation,
// take the first ~5 significant words, join with hyphens. De-duplicates
// against existing by appending -2, -3, etc. on collision.
func GenerateSlug(text string, existing []string) string {
	lower := strings.ToLower(text)
	words := slugWordRe.FindAllString(lower, -1)

	// Filter out very short/common words.
	var significant []string
	for _, w := range words {
		if len(w) >= 2 {
			significant = append(significant, w)
		}
	}
	if len(significant) == 0 {
		significant = words
	}
	if len(significant) > 5 {
		significant = significant[:5]
	}

	base := strings.Join(significant, "-")
	if base == "" {
		base = "law"
	}

	// De-duplicate.
	seen := map[string]bool{}
	for _, s := range existing {
		seen[s] = true
	}
	if !seen[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !seen[candidate] {
			return candidate
		}
	}
}

// IsValidSlug reports whether s is a valid law slug per the charset spec
// (docs/linking.md §3.1): lowercase letter followed by lowercase
// alphanumerics and hyphens.
func IsValidSlug(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case i == 0 && r >= 'a' && r <= 'z':
			// ok
		case i > 0 && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-'):
			// ok
		default:
			return false
		}
	}
	return true
}

// isWordRune reports whether r is a letter or digit (used for word
// boundary detection in slug generation).
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
