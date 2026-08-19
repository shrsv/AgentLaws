// Package resolver resolves citations (e.g. "2.5.3"), section IDs, and
// law slugs to their source location within a compiled Lawbook.
// See docs/PLAN1.md §15 and docs/linking.md §2.
package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
)

// ErrNotFound is returned when a citation or ID does not exist in the
// lawbook. CLI commands map this to exit code 3 (PLAN1 §32).
var ErrNotFound = errors.New("not found")

// Kind distinguishes what a resolved reference points at.
type Kind int

const (
	KindLaw Kind = iota
	KindSection
)

// Resolved is the result of resolving one reference token.
type Resolved struct {
	Kind    Kind
	Law     model.Law     // valid iff Kind == KindLaw
	Section model.Section // valid iff Kind == KindSection
}

// AnchorFor returns the stable anchor string for a resolved reference.
// For laws with a slug this is "<section-id>.<slug>"; for laws without a
// slug it falls back to the citation number. For sections it is the
// section ID.
func AnchorFor(r Resolved) string {
	switch r.Kind {
	case KindLaw:
		if r.Law.Slug != "" {
			return r.Law.SectionID + "." + r.Law.Slug
		}
		return r.Law.Number
	case KindSection:
		return r.Section.ID
	}
	panic("unreachable")
}

// Resolve resolves token against book, trying — in this exact order — every
// addressing form AgentLaws supports. See docs/linking.md §2.1 for the full
// specification and the rationale for this precedence order.
func Resolve(book model.Lawbook, token string) (Resolved, error) {
	// (a) Exact Section.ID match — highest precedence.
	for _, s := range book.Sections {
		if s.ID == token {
			return Resolved{Kind: KindSection, Section: s}, nil
		}
	}

	// (b) Fully-qualified law identity: "<section-id>.<law-slug>". Split at
	// the LAST '.' — section ids may themselves contain multiple dots, but
	// a law slug never contains a '.', so the last dot is always the
	// correct split point.
	if lastDot := strings.LastIndex(token, "."); lastDot != -1 {
		sectionPart, slugPart := token[:lastDot], token[lastDot+1:]
		for _, s := range book.Sections {
			if s.ID != sectionPart {
				continue
			}
			for _, l := range s.Laws {
				if l.Slug == slugPart {
					return Resolved{Kind: KindLaw, Law: l}, nil
				}
			}
		}
	}

	// (c) Law citation number, e.g. "2.5.3" — legacy/as-compiled form.
	for _, s := range book.Sections {
		for _, l := range s.Laws {
			if l.Number == token {
				return Resolved{Kind: KindLaw, Law: l}, nil
			}
		}
	}

	// (d) Section presentation number, e.g. "2.5".
	for _, s := range book.Sections {
		if s.Number == token {
			return Resolved{Kind: KindSection, Section: s}, nil
		}
	}

	// (e) Bare law slug, unqualified — only if unambiguous lawbook-wide.
	var match *model.Law
	ambiguous := false
	for _, s := range book.Sections {
		for i, l := range s.Laws {
			if l.Slug == "" || l.Slug != token {
				continue
			}
			if match != nil {
				ambiguous = true
			}
			match = &s.Laws[i]
		}
	}
	if match != nil && !ambiguous {
		return Resolved{Kind: KindLaw, Law: *match}, nil
	}
	if ambiguous {
		return Resolved{}, fmt.Errorf("%w: %q is an ambiguous bare slug (used in more than one section) — use the fully-qualified <section-id>.<slug> form", ErrNotFound, token)
	}

	return Resolved{}, fmt.Errorf("%w: %q", ErrNotFound, token)
}

// ResolveLaw resolves a canonical citation such as "2.5.3" or a law slug
// to its Law.
func ResolveLaw(book model.Lawbook, citation string) (model.Law, error) {
	for _, s := range book.Sections {
		for _, l := range s.Laws {
			if l.Number == citation || l.Slug == citation {
				return l, nil
			}
		}
	}
	return model.Law{}, fmt.Errorf("%w: law %q", ErrNotFound, citation)
}

// ResolveSection resolves a stable section ID (e.g. "engineering.security")
// or a canonical section number (e.g. "2.5") to its Section.
func ResolveSection(book model.Lawbook, id string) (model.Section, error) {
	for _, s := range book.Sections {
		if s.ID == id || s.Number == id {
			return s, nil
		}
	}
	return model.Section{}, fmt.Errorf("%w: section %q", ErrNotFound, id)
}
