// Package provenance collects Git metadata, computes content hashes, and
// builds the provenance manifest for a compiled lawbook. See docs/PLAN1.md
// §13, §24-§25, §34, §36-§39.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/shrsv/AgentLaws/internal/git"
	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/internal/version"
)

// Collect assembles provenance for the lawbook at path: the alaws binary's
// own version/build time always, plus - when path is inside a Git
// repository - the compiling identity, HEAD revision, last-commit author,
// and whether the book's own files are dirty relative to HEAD. A path
// outside any Git repository is not an error: only the Git-derived fields
// are left empty (docs/PLAN1.md §47 - compilation stays deterministic and
// available regardless of VCS availability).
func Collect(path string) (model.Provenance, error) {
	v, buildTime := version.Info()
	prov := model.Provenance{
		CompiledAt:         time.Now().Format(time.RFC3339),
		AgentLawsVersion:   v,
		AgentLawsBuildTime: buildTime,
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return prov, err
	}

	root, err := git.RepoRoot(absPath)
	if err != nil {
		if errors.Is(err, git.ErrNotAGitRepo) {
			return prov, nil
		}
		return prov, err
	}

	if name, email, err := git.Identity(absPath); err == nil {
		prov.CompilerName, prov.CompilerEmail = name, email
	}
	if rev, err := git.HeadRevision(absPath); err == nil {
		prov.Revision = rev
	}
	if author, date, err := git.LastCommitInfo(absPath); err == nil {
		prov.HeadCommitAuthor, prov.HeadCommitDate = author, date
	}
	if dirty, hash, err := git.WorkingTreeState(root, absPath); err == nil {
		prov.Dirty, prov.WorkingTreeHash = dirty, hash
	}

	return prov, nil
}

// canonicalPayload is what gets content-hashed and signed: semantic
// content only. Provenance is deliberately excluded - it changes on every
// single compile (timestamp, possibly working-tree hash), and including it
// would make the signature change even when no law changed, defeating the
// point of signing (docs/PLAN1.md §47, §25's determinism invariant).
type canonicalPayload struct {
	Metadata model.LawbookMetadata
	Sections []model.Section
}

func canonicalize(book model.Lawbook) ([]byte, error) {
	return json.Marshal(canonicalPayload{Metadata: book.Metadata, Sections: book.Sections})
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashLawbook returns the content hash of book's semantic content
// (Metadata + Sections), excluding Provenance (docs/PLAN1.md §36). This is
// the value that gets signed.
func HashLawbook(book model.Lawbook) (string, error) {
	b, err := canonicalize(book)
	if err != nil {
		return "", err
	}
	return hashBytes(b), nil
}

// HashSection returns the content hash of one section (id, title, level,
// commentary, and its laws) - useful for detecting whether a particular
// section changed independent of the rest of the lawbook (docs/PLAN1.md
// §36).
func HashSection(s model.Section) (string, error) {
	b, err := json.Marshal(struct {
		ID         string
		Title      string
		Level      int
		Commentary string
		Laws       []model.Law
	}{s.ID, s.Title, s.Level, s.Commentary, s.Laws})
	if err != nil {
		return "", err
	}
	return hashBytes(b), nil
}

// HashLaw returns the content hash of one law's stable identity + text
// (SectionID + Index + Text - never Number, which is presentation-only and
// shifts on reorder; docs/PLAN1.md §14, §36).
func HashLaw(l model.Law) (string, error) {
	b, err := json.Marshal(struct {
		SectionID string
		Index     int
		Text      string
	}{l.SectionID, l.Index, l.Text})
	if err != nil {
		return "", err
	}
	return hashBytes(b), nil
}

// LawChange describes one law whose text differs between two compiled
// Lawbooks, matched by its stable identity (SectionID + Index), not its
// presentation citation number.
type LawChange struct {
	SectionID string
	Index     int
	OldNumber string
	NewNumber string
	OldText   string
	NewText   string
}

// LawbookDiff is the result of comparing two compilations of the same
// lawbook, matching sections and laws by stable identity rather than
// presentation numbering (docs/PLAN1.md §38).
type LawbookDiff struct {
	AddedSections   []string // section IDs present in new but not old
	RemovedSections []string // section IDs present in old but not new
	AddedLaws       []model.Law
	RemovedLaws     []model.Law
	ModifiedLaws    []LawChange
}

type lawKey struct {
	SectionID string
	Index     int
}

// Diff compares old and new compilations of the same lawbook (typically
// two revisions of the same source), matching sections by ID and laws by
// SectionID+Index - the stable identity from §14 - so a reorder that only
// changes presentation numbers is never reported as an add/remove
// (docs/PLAN1.md §38).
func Diff(old, new model.Lawbook) LawbookDiff {
	oldSections := map[string]model.Section{}
	for _, s := range old.Sections {
		oldSections[s.ID] = s
	}
	newSections := map[string]model.Section{}
	for _, s := range new.Sections {
		newSections[s.ID] = s
	}

	var diff LawbookDiff

	for id := range newSections {
		if _, ok := oldSections[id]; !ok {
			diff.AddedSections = append(diff.AddedSections, id)
		}
	}
	for id := range oldSections {
		if _, ok := newSections[id]; !ok {
			diff.RemovedSections = append(diff.RemovedSections, id)
		}
	}

	oldLaws := map[lawKey]model.Law{}
	for _, s := range old.Sections {
		for _, l := range s.Laws {
			oldLaws[lawKey{s.ID, l.Index}] = l
		}
	}
	newLaws := map[lawKey]model.Law{}
	for _, s := range new.Sections {
		for _, l := range s.Laws {
			newLaws[lawKey{s.ID, l.Index}] = l
		}
	}

	for k, nl := range newLaws {
		ol, ok := oldLaws[k]
		if !ok {
			diff.AddedLaws = append(diff.AddedLaws, nl)
			continue
		}
		if ol.Text != nl.Text {
			diff.ModifiedLaws = append(diff.ModifiedLaws, LawChange{
				SectionID: k.SectionID,
				Index:     k.Index,
				OldNumber: ol.Number,
				NewNumber: nl.Number,
				OldText:   ol.Text,
				NewText:   nl.Text,
			})
		}
	}
	for k, ol := range oldLaws {
		if _, ok := newLaws[k]; !ok {
			diff.RemovedLaws = append(diff.RemovedLaws, ol)
		}
	}

	return diff
}

// Manifest is the machine-readable provenance manifest described in
// PLAN1 §24: a content hash of the compiled lawbook's semantic content,
// the Provenance describing how/when/by-whom/with-what-tool it was
// compiled, and - once `alaws sign` has run - a Signature over ContentHash.
type Manifest struct {
	Lawbook     string           `json:"lawbook"`
	ContentHash string           `json:"content_hash"`
	Provenance  model.Provenance `json:"provenance"`
	Signature   string           `json:"signature,omitempty"`
}

// BuildManifest constructs the manifest for a compiled Lawbook. book's
// Provenance field must already be populated (pkg/alaws.Compile/Load do
// this on every compile - see docs/PLAN1.md's provenance plan step 5).
func BuildManifest(book model.Lawbook) (Manifest, error) {
	hash, err := HashLawbook(book)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Lawbook:     book.Metadata.Title,
		ContentHash: hash,
		Provenance:  book.Provenance,
	}, nil
}

// LawHistory is the change history of a single law, resolved through Git
// line-range history on its current source location (docs/PLAN1.md
// §37-§39).
type LawHistory struct {
	Citation      string
	Introduced    string // commit hash, "" if unavailable
	Modifications []git.HistoryEntry
}

// History returns the Git history of the law identified by citation.
func History(book model.Lawbook, citation string) (LawHistory, error) {
	law, err := resolver.ResolveLaw(book, citation)
	if err != nil {
		return LawHistory{}, err
	}

	absPath, err := filepath.Abs(law.Source.Path)
	if err != nil {
		return LawHistory{}, err
	}

	root, err := git.RepoRoot(filepath.Dir(absPath))
	if err != nil {
		return LawHistory{}, fmt.Errorf("history for %s: %w", citation, err)
	}

	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return LawHistory{}, err
	}

	entries, err := git.LineHistory(root, rel, law.Source.LineStart, law.Source.LineEnd)
	if err != nil {
		return LawHistory{}, err
	}

	hist := LawHistory{Citation: citation, Modifications: entries}
	if len(entries) > 0 {
		hist.Introduced = entries[len(entries)-1].Commit
	}
	return hist, nil
}
