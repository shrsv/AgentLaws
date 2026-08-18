// Package resolver resolves citations (e.g. "2.5.3") and section IDs to
// their source location within a compiled Lawbook. See docs/PLAN1.md §15,
// §34.
package resolver

import (
	"errors"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned by every stub in this package until
// resolution is implemented per PLAN1 §64 Milestone 2.
var ErrNotImplemented = errors.New("resolver: not implemented")

// ErrNotFound is returned when a citation or ID does not exist in the
// lawbook. CLI commands should map this to exit code 3 (PLAN1 §32).
var ErrNotFound = errors.New("resolver: not found")

// ResolveLaw resolves a canonical citation such as "2.5.3" to its Law.
func ResolveLaw(book model.Lawbook, citation string) (model.Law, error) {
	return model.Law{}, ErrNotImplemented
}

// ResolveSection resolves a stable section ID such as "engineering.security"
// to its Section.
func ResolveSection(book model.Lawbook, id string) (model.Section, error) {
	return model.Section{}, ErrNotImplemented
}
