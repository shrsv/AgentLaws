package alaws

import (
	"fmt"
	"strings"

	"github.com/shrsv/AgentLaws/internal/git"
)

// CommitDetail is the full context of one Git commit within a book's
// history: the commit metadata, the files it touched, and the semantic
// diff (sections/laws added/removed/modified). The Diff field is nil
// when the commit cannot be compiled (e.g. the book didn't exist yet).
type CommitDetail struct {
	Commit  string           `json:"commit"`
	Author  string           `json:"author"`
	Date    string           `json:"date"`
	Summary string           `json:"summary"`
	Files   []git.FileChange `json:"files"`
	Diff    *LawbookDiff     `json:"diff,omitempty"`
}

// CommitDetail returns the full detail of a single commit: the files it
// touched within the book, plus the semantic diff (which sections/laws
// changed) computed by compiling the book at the commit and its parent.
func CommitDetailFor(bookPath, commit string) (CommitDetail, error) {
	root, rel, err := repoRel(bookPath)
	if err != nil {
		return CommitDetail{}, err
	}

	// Resolve the commit to a full hash and get its metadata.
	fullHash, err := git.ResolveRevision(root, commit)
	if err != nil {
		return CommitDetail{}, err
	}

	entries, err := git.Log(root, []string{rel}, 0)
	if err != nil {
		return CommitDetail{}, err
	}
	var entry git.CommitInfo
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Commit, fullHash) || strings.HasPrefix(fullHash, e.Commit) {
			entry = e
			found = true
			break
		}
	}
	if !found {
		return CommitDetail{}, fmt.Errorf("commit %s not found in book history", commit)
	}

	// Get file-level changes.
	files, err := git.DiffFiles(root, commit+"~1", commit, []string{rel})
	if err != nil {
		files = nil // non-fatal: first commit has no parent
	}

	// Get semantic diff by compiling at this commit and its parent.
	var diff *LawbookDiff
	newBook, errNew := CompileRevision(bookPath, commit)
	oldBook, errOld := CompileRevision(bookPath, commit+"~1")
	if errNew == nil && errOld == nil {
		d := Diff(oldBook, newBook)
		diff = &d
	}

	return CommitDetail{
		Commit:  entry.Commit,
		Author:  entry.Author,
		Date:    entry.Date,
		Summary: entry.Summary,
		Files:   files,
		Diff:    diff,
	}, nil
}
