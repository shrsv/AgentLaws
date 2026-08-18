// Package resolver resolves citations (e.g. "2.5.3") and section IDs to
// their source location within a compiled Lawbook. See docs/PLAN1.md §15.
package resolver

import (
	"errors"
	"fmt"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotFound is returned when a citation or ID does not exist in the
// lawbook. CLI commands map this to exit code 3 (PLAN1 §32).
var ErrNotFound = errors.New("not found")

// ResolveLaw resolves a canonical citation such as "2.5.3" to its Law.
func ResolveLaw(book model.Lawbook, citation string) (model.Law, error) {
	for _, s := range book.Sections {
		for _, l := range s.Laws {
			if l.Number == citation {
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
