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
//
// Slug, when present, is the law's stable identity combined with its
// section's ID (see docs/linking.md §2). Authors assign it via {#slug}
// in the source Markdown. Falls back to Number when absent.
type Law struct {
	Number    string
	Index     int
	Text      string
	Slug      string
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

// Provenance describes who/what produced a compiled Lawbook, from where,
// and with what tool - populated on every compile, not only when a book is
// explicitly signed. See docs/PLAN1.md §24-§25 and the "enriched
// model.Provenance" design in the provenance/history/signing plan.
type Provenance struct {
	Revision           string // HEAD commit hash ("" if not a Git repo)
	Dirty              bool   // true if the book's own files differ from HEAD
	WorkingTreeHash    string // hash of the uncommitted diff + untracked content; "" when clean
	CompiledAt         string // RFC3339, local timezone offset
	CompilerName       string // git config user.name - who ran this compile
	CompilerEmail      string // git config user.email
	HeadCommitAuthor   string // HEAD's author "Name <email>" - who made the last commit
	HeadCommitDate     string // HEAD's author date, RFC3339 with offset
	AgentLawsVersion   string // alaws binary version
	AgentLawsBuildTime string // when that alaws binary was built (best-effort, may be "")
}

// Lawbook is the compiled representation of one lawbook cluster.
type Lawbook struct {
	Metadata   LawbookMetadata
	Sections   []Section
	Provenance Provenance
}
