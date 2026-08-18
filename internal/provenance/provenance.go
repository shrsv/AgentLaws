// Package provenance collects Git metadata and constructs the provenance
// manifest for a compiled lawbook. See docs/PLAN1.md §13, §24-§25, §34.
package provenance

import (
	"errors"

	"github.com/shrsv/AgentLaws/internal/model"
)

// ErrNotImplemented is returned by every stub in this package until
// provenance collection is implemented per PLAN1 §64 Milestone 6.
var ErrNotImplemented = errors.New("provenance: not implemented")

// Collect gathers Git identity and revision information for the repository
// containing path.
func Collect(path string) (model.Provenance, error) {
	return model.Provenance{}, ErrNotImplemented
}

// Manifest is the machine-readable provenance manifest described in
// PLAN1 §24.
type Manifest struct {
	Lawbook          string           `json:"lawbook"`
	Revision         string           `json:"revision"`
	CompiledAt       string           `json:"compiled_at"`
	AgentLawsVersion string           `json:"agentlaws_version"`
	Compiler         CompilerIdentity `json:"compiler"`
	Signature        string           `json:"signature,omitempty"`
}

// CompilerIdentity identifies who compiled a lawbook.
type CompilerIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// BuildManifest constructs the canonical manifest for a compiled Lawbook.
func BuildManifest(book model.Lawbook) (Manifest, error) {
	return Manifest{}, ErrNotImplemented
}

// LawHistory is the change history of a single law, resolved through its
// stable section identity plus clause index rather than its (presentation)
// citation number. See docs/PLAN1.md §37-§39.
type LawHistory struct {
	Citation      string
	Introduced    string // commit hash
	Modifications []HistoryEntry
}

// HistoryEntry is one Git commit that touched a law's text.
type HistoryEntry struct {
	Commit  string
	Author  string
	Date    string
	Summary string
}

// History returns the Git history of the law identified by citation.
func History(book model.Lawbook, citation string) (LawHistory, error) {
	return LawHistory{}, ErrNotImplemented
}
