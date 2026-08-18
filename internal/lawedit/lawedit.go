// Package lawedit locates the `<!-- alaws:laws -->` region in a section
// file and edits its numbered list without disturbing surrounding Markdown.
// It backs `alaws law add`/`alaws law remove` (PLAN1 §32). This is flagged
// in docs/PLAN1.md as the highest-risk CLI mutation, since it edits
// structured Markdown in place rather than a config file.
package lawedit

import "errors"

// ErrNotImplemented is returned by every stub in this package until law
// editing is implemented per PLAN1 §64 Milestone 4.
var ErrNotImplemented = errors.New("lawedit: not implemented")

// Add appends a new numbered clause to the laws region of the section file
// at path. If after > 0, the clause is inserted immediately after that
// existing clause number instead of at the end.
func Add(path string, text string, after int) error {
	return ErrNotImplemented
}

// Remove deletes the numbered clause `number` from the laws region of the
// section file at path, renumbering subsequent clauses.
func Remove(path string, number int, force bool) error {
	return ErrNotImplemented
}
