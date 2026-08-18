// Package git collects Git metadata for provenance (docs/PLAN1.md §13, §25,
// §37-39): identity, revision, working-tree state, and line-range history.
// It shells out to the system `git` binary rather than vendoring a Git
// implementation - see docs/PLAN1.md §39 ("Git remains the historical
// source of truth; AgentLaws adds structure to Git history, it does not
// replace it").
package git

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ErrNotAGitRepo is returned when path isn't inside a Git working tree.
// Callers treat this as "provenance unavailable," not a failure - a
// lawbook must still compile outside Git (docs/PLAN1.md §47 "compilation
// is deterministic" applies regardless of VCS availability).
var ErrNotAGitRepo = errors.New("git: not a git repository")

const fieldSep = "\x1f"

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") {
			return "", ErrNotAGitRepo
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// RepoRoot returns the top-level directory of the Git working tree
// containing path.
func RepoRoot(path string) (string, error) {
	return run(path, "rev-parse", "--show-toplevel")
}

// Identity returns the local Git identity (git config user.name/user.email)
// used to attribute a compile/sign to the person who ran it.
func Identity(path string) (name, email string, err error) {
	name, err = run(path, "config", "user.name")
	if err != nil {
		return "", "", err
	}
	email, err = run(path, "config", "user.email")
	if err != nil {
		return "", "", err
	}
	return name, email, nil
}

// HeadRevision returns the full HEAD commit hash for the repository
// containing path.
func HeadRevision(path string) (string, error) {
	return run(path, "rev-parse", "HEAD")
}

// ResolveRevision resolves rev (a commit-ish: a hash, tag, branch, or
// relative ref like "HEAD~3") to its full commit hash.
func ResolveRevision(repoRoot, rev string) (string, error) {
	return run(repoRoot, "rev-parse", rev)
}

// LastCommitInfo returns the author ("Name <email>") and author date
// (RFC3339 with offset) of HEAD - the person who made the last commit,
// which may differ from Identity (whoever is compiling right now).
func LastCommitInfo(path string) (author, date string, err error) {
	out, err := run(path, "log", "-1", "--format=%an <%ae>"+fieldSep+"%aI")
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(out, fieldSep, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("git log -1: unexpected output %q", out)
	}
	return parts[0], parts[1], nil
}

// WorkingTreeState reports whether scopePath (a subdirectory of repoRoot,
// typically a lawbook's own directory) differs from HEAD, and - if so - a
// content hash of exactly what's uncommitted (the diff against HEAD plus
// the content of any untracked files), so two dirty compiles of different
// in-progress edits are distinguishable rather than both just "dirty".
// hash is "" when the tree is clean.
func WorkingTreeState(repoRoot, scopePath string) (dirty bool, hash string, err error) {
	rel, err := filepath.Rel(repoRoot, scopePath)
	if err != nil {
		return false, "", err
	}
	if rel == "." {
		rel = ""
	}

	h := sha256.New()

	if _, headErr := run(repoRoot, "rev-parse", "--verify", "-q", "HEAD"); headErr != nil {
		// No commits yet - everything under scopePath is uncommitted by
		// definition. Hash the tree directly rather than diffing against a
		// HEAD that doesn't exist.
		anyFile := false
		walkErr := filepath.WalkDir(scopePath, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			anyFile = true
			relPath, _ := filepath.Rel(scopePath, p)
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return readErr
			}
			fmt.Fprintf(h, "%s\x00", relPath)
			h.Write(data)
			return nil
		})
		if walkErr != nil {
			return false, "", walkErr
		}
		if !anyFile {
			return false, "", nil
		}
		return true, hex.EncodeToString(h.Sum(nil)), nil
	}

	diffArgs := []string{"diff", "HEAD", "--"}
	if rel != "" {
		diffArgs = append(diffArgs, rel)
	}
	diffOut, err := run(repoRoot, diffArgs...)
	if err != nil {
		return false, "", err
	}
	h.Write([]byte(diffOut))

	untrackedArgs := []string{"ls-files", "--others", "--exclude-standard", "--"}
	if rel != "" {
		untrackedArgs = append(untrackedArgs, rel)
	} else {
		untrackedArgs = append(untrackedArgs, ".")
	}
	untrackedOut, err := run(repoRoot, untrackedArgs...)
	if err != nil {
		return false, "", err
	}
	var untracked []string
	if untrackedOut != "" {
		untracked = strings.Split(untrackedOut, "\n")
		sort.Strings(untracked)
	}
	for _, f := range untracked {
		data, readErr := os.ReadFile(filepath.Join(repoRoot, f))
		if readErr != nil {
			continue
		}
		fmt.Fprintf(h, "%s\x00", f)
		h.Write(data)
	}

	dirty = diffOut != "" || len(untracked) > 0
	if !dirty {
		return false, "", nil
	}
	return true, hex.EncodeToString(h.Sum(nil)), nil
}

// CommitInfo is one Git commit touching some path(s), used for
// book-level traceability (`alaws log`, docs/PLAN1.md §37).
type CommitInfo struct {
	Commit  string
	Author  string
	Date    string // RFC3339 with offset
	Summary string
}

// Log returns commits touching any of paths (repoRoot-relative or
// absolute), most recent first, up to limit commits (0 = no limit).
func Log(repoRoot string, paths []string, limit int) ([]CommitInfo, error) {
	args := []string{"log", "--format=%H" + fieldSep + "%an <%ae>" + fieldSep + "%aI" + fieldSep + "%s"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}
	args = append(args, "--")
	args = append(args, paths...)

	out, err := run(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var entries []CommitInfo
	for line := range strings.SplitSeq(out, "\n") {
		parts := strings.SplitN(line, fieldSep, 4)
		if len(parts) != 4 {
			continue
		}
		entries = append(entries, CommitInfo{Commit: parts[0], Author: parts[1], Date: parts[2], Summary: parts[3]})
	}
	return entries, nil
}

// Archive extracts the tree at rev's relPath (repoRoot-relative) into
// destDir, stripping relPath's own prefix so destDir directly contains
// relPath's contents - e.g. Archive(root, "HEAD~3", "examples/eng", tmp)
// populates tmp/alaws.toml, tmp/security/secrets.md, etc, ready to be
// compiled as a lawbook on its own (used to compile a past revision
// without disturbing the working tree, for `alaws log`/`alaws diff`).
func Archive(repoRoot, rev, relPath, destDir string) error {
	cmd := exec.Command("git", "-C", repoRoot, "archive", "--format=tar", rev, "--", relPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	prefix := ""
	if relPath != "." && relPath != "" {
		prefix = relPath + "/"
	}

	tr := tar.NewReader(stdout)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = cmd.Wait()
			return err
		}
		if prefix != "" && !strings.HasPrefix(hdr.Name, prefix) {
			continue
		}
		name := strings.TrimPrefix(hdr.Name, prefix)
		if name == "" {
			continue
		}
		target := filepath.Join(destDir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git archive: %s", msg)
	}
	return nil
}

// HistoryEntry is one Git commit that touched a line range.
type HistoryEntry struct {
	Commit  string
	Author  string
	Date    string // RFC3339 with offset
	Summary string
}

// FileChange describes one file that changed between two commits.
type FileChange struct {
	Path    string
	Status  string // "A", "M", "D", "R" (added, modified, deleted, renamed)
	Added   int    // lines added (0 for binary or status D)
	Deleted int    // lines deleted (0 for binary or status D)
}

// DiffFiles returns the list of files that changed between two revisions,
// scoped to paths (repoRoot-relative). It uses git diff --numstat for
// structured line-count data.
func DiffFiles(repoRoot, from, to string, paths []string) ([]FileChange, error) {
	args := []string{"diff", "--numstat", "--diff-filter=ADMRT", from, to, "--"}
	args = append(args, paths...)
	out, err := run(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var changes []FileChange
	for line := range strings.SplitSeq(out, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		added := 0
		deleted := 0
		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &added)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &deleted)
		}
		changes = append(changes, FileChange{
			Path:    parts[2],
			Status:  diffStatus(added, deleted, parts[0], parts[1]),
			Added:   added,
			Deleted: deleted,
		})
	}
	return changes, nil
}

func diffStatus(added, deleted int, rawAdd, rawDel string) string {
	if rawAdd == "-" && rawDel == "-" {
		return "B" // binary
	}
	if added > 0 && deleted == 0 {
		return "A"
	}
	if added == 0 && deleted > 0 {
		return "D"
	}
	return "M"
}

var commitHeaderRe = regexp.MustCompile(`^[0-9a-f]{40}` + fieldSep)

// LineHistory returns the Git history of the line range [lineStart,
// lineEnd] in relPath (relative to repoRoot), newest first, via `git log
// -L` - the line range is followed through the file's edits automatically
// (docs/PLAN1.md §37). Note: --follow (tracking the file across renames)
// is deliberately not combined with -L here - Git rejects that combination
// ("--follow requires exactly one pathspec") in the versions this was
// tested against; a renamed file's pre-rename history is simply not
// included.
func LineHistory(repoRoot, relPath string, lineStart, lineEnd int) ([]HistoryEntry, error) {
	format := "%H" + fieldSep + "%an <%ae>" + fieldSep + "%aI" + fieldSep + "%s"
	out, err := run(repoRoot, "log",
		fmt.Sprintf("--format=%s", format),
		fmt.Sprintf("-L%d,%d:%s", lineStart, lineEnd, relPath))
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var entries []HistoryEntry
	for line := range strings.SplitSeq(out, "\n") {
		if !commitHeaderRe.MatchString(line) {
			continue // diff-hunk body line, not a commit header
		}
		parts := strings.SplitN(line, fieldSep, 4)
		if len(parts) != 4 {
			continue
		}
		entries = append(entries, HistoryEntry{
			Commit:  parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Summary: parts[3],
		})
	}
	return entries, nil
}
