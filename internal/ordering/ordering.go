// Package ordering is the single code path that reads and writes the
// `ordering` list in alaws.toml. Both the CLI (`alaws chapter`/`section`,
// PLAN1 §32) and the future drag-and-drop UI (PLAN1 §29) call this package
// rather than editing TOML themselves, so there is exactly one place that
// mutates ordering (PLAN1 §30, §52).
package ordering

import "errors"

// ErrNotImplemented is returned by every stub in this package until
// ordering mutation is implemented per PLAN1 §64 Milestone 4.
var ErrNotImplemented = errors.New("ordering: not implemented")

// Placement describes where a new or moved entry should be inserted
// relative to the existing ordering.
type Placement struct {
	After    string // insert immediately after this entry's path/id, if set
	Position int    // 1-based absolute position, used if After is empty and > 0
}

// Node is one entry in the ordering, resolved to its level and derived
// parent, per the outline rule in PLAN1 §32.
type Node struct {
	Path     string
	ID       string
	Level    int
	ParentID string // "" for a chapter (top-level, Level == 1)
}

// Tree computes the chapter/section parent-child structure implied by a
// flat ordering list plus each entry's Level.
func Tree(configPath string) ([]Node, error) {
	return nil, ErrNotImplemented
}

// Insert adds a new ordering entry at the position described by placement
// and rewrites alaws.toml in place.
func Insert(configPath string, entryPath string, placement Placement) error {
	return ErrNotImplemented
}

// Move relocates an existing ordering entry (and, for a chapter, its
// descendants) to a new position and rewrites alaws.toml in place.
func Move(configPath string, entryPath string, placement Placement) error {
	return ErrNotImplemented
}

// Remove deletes an ordering entry from alaws.toml. It returns an error if
// the entry is a chapter with descendants unless force is true.
func Remove(configPath string, entryPath string, force bool) error {
	return ErrNotImplemented
}

// SectionMeta is the frontmatter for a newly created chapter or section
// file (PLAN1 §6).
type SectionMeta struct {
	Title string
	ID    string
	Level int
}

// NewBook creates a new alaws.toml at path with the given title and an
// empty ordering, establishing a new lawbook cluster (PLAN1 §4).
func NewBook(path string, title string) error {
	return ErrNotImplemented
}

// NewSectionFile writes a new section Markdown file at path with meta's
// frontmatter and an empty commentary/laws skeleton (PLAN1 §6), ready to be
// added to a book's ordering via Insert.
func NewSectionFile(path string, meta SectionMeta) error {
	return ErrNotImplemented
}
