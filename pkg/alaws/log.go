package alaws

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shrsv/AgentLaws/internal/git"
)

// LogEntry is one commit that touched a book's own files, used for
// book-level traceability (docs/PLAN1.md §37, `alaws log`).
type LogEntry = git.CommitInfo

// Log returns commits touching the book at path, most recent first, up to
// limit commits (0 = no limit).
func Log(path string, limit int) ([]LogEntry, error) {
	root, rel, err := repoRel(path)
	if err != nil {
		return nil, err
	}
	return git.Log(root, []string{rel}, limit)
}

// CompileRevision compiles the lawbook at path as it existed at Git
// revision rev, by materializing that revision's tree into a temporary
// directory (leaving the working tree untouched) and compiling it there.
// The returned Book's Provenance.Revision is set to rev's resolved hash,
// rather than reflecting the temporary directory it was actually compiled
// from.
func CompileRevision(path, rev string) (*Book, error) {
	root, rel, err := repoRel(path)
	if err != nil {
		return nil, err
	}

	tmp, err := os.MkdirTemp("", "alaws-revision-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	if err := git.Archive(root, rev, rel, tmp); err != nil {
		return nil, fmt.Errorf("compiling %s at %s: %w", path, rev, err)
	}

	book, err := Compile(tmp)
	if fullRev, revErr := git.ResolveRevision(root, rev); revErr == nil {
		book.lawbook.Provenance.Revision = fullRev
		book.lawbook.Provenance.Dirty = false
		book.lawbook.Provenance.WorkingTreeHash = ""
	}
	return book, err
}

// repoRel resolves path to its containing repo's root and path's location
// relative to that root.
func repoRel(path string) (root, rel string, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	root, err = git.RepoRoot(absPath)
	if err != nil {
		return "", "", err
	}
	rel, err = filepath.Rel(root, absPath)
	return root, rel, err
}
