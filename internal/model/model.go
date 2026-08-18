// Package model defines the Lawbook intermediate representation (IR).
//
// Renderers, resolvers, and the CLI all operate on this IR rather than on
// Markdown or TOML directly. See docs/PLAN1.md §12-§14.
package model

// SourceRef locates a section or law in the underlying source tree.
type SourceRef struct {
	Path      string
	LineStart int
	LineEnd   int
}

// Law is a single numbered clause within a section's laws region.
//
// Number is the canonical citation (e.g. "2.5.3"), assigned during
// compilation. It is never authored directly and must not be treated as a
// stable identity — SectionID + Index is the stable identity (§14).
type Law struct {
	Number    string
	Index     int
	Text      string
	SectionID string
	Source    SourceRef
}

// Section corresponds to one source Markdown file.
//
// ID is the stable identity (§14). Number is the canonical presentation
// number assigned during compilation (e.g. "2.5") and may change between
// compilations if the lawbook is reordered.
type Section struct {
	ID         string
	Number     string
	Title      string
	Level      int
	ParentID   string // derived: nearest preceding section with Level < this Level ("" for chapters)
	Source     SourceRef
	Commentary string
	Laws       []Law
}

// LawbookMetadata holds the fields sourced from alaws.toml.
type LawbookMetadata struct {
	Title    string
	Ordering []string
}

// Provenance describes who/what produced a compiled Lawbook and from where.
// See docs/PLAN1.md §24-§25.
type Provenance struct {
	Revision         string
	CompiledAt       string
	AgentLawsVersion string
	CompilerName     string
	CompilerEmail    string
	Signature        string
}

// Lawbook is the compiled representation of one lawbook cluster.
type Lawbook struct {
	Metadata   LawbookMetadata
	Sections   []Section
	Provenance Provenance
}
