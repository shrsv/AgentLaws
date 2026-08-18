package git

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests run against the AgentLaws repo itself (this file lives inside
// it). They verify the git package's real behavior against actual git history.

func repoRootOrSkip(t *testing.T) string {
	t.Helper()
	// Walk up from this test file to find the repo root.
	dir, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get working directory")
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("not inside a git repository")
		}
		dir = parent
	}
}

func TestIdentity(t *testing.T) {
	root := repoRootOrSkip(t)
	name, email, err := Identity(root)
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}
	if name == "" {
		t.Fatal("empty user.name")
	}
	if email == "" {
		t.Fatal("empty user.email")
	}
	t.Logf("identity: %s <%s>", name, email)
}

func TestHeadRevision(t *testing.T) {
	root := repoRootOrSkip(t)
	rev, err := HeadRevision(root)
	if err != nil {
		t.Fatalf("HeadRevision: %v", err)
	}
	if len(rev) < 7 {
		t.Fatalf("HEAD too short: %q", rev)
	}
	t.Logf("HEAD: %s", rev)
}

func TestLastCommitInfo(t *testing.T) {
	root := repoRootOrSkip(t)
	author, date, err := LastCommitInfo(root)
	if err != nil {
		t.Fatalf("LastCommitInfo: %v", err)
	}
	if author == "" {
		t.Fatal("empty author")
	}
	if date == "" {
		t.Fatal("empty date")
	}
	t.Logf("last commit: %s at %s", author, date)
}

func TestRepoRoot(t *testing.T) {
	root := repoRootOrSkip(t)
	// Subdirectory should resolve to same root
	sub := filepath.Join(root, "internal")
	r, err := RepoRoot(sub)
	if err != nil {
		t.Fatalf("RepoRoot(%s): %v", sub, err)
	}
	if r != root {
		t.Fatalf("expected %s, got %s", root, r)
	}
}

func TestWorkingTreeState(t *testing.T) {
	root := repoRootOrSkip(t)
	dirty, hash, err := WorkingTreeState(root, root)
	if err != nil {
		t.Fatalf("WorkingTreeState: %v", err)
	}
	t.Logf("dirty=%v hash=%q", dirty, hash)
	// We don't assert dirty/clean since the test environment may vary,
	// but hash should be empty when clean and non-empty when dirty.
	if dirty && hash == "" {
		t.Fatal("dirty but no working tree hash")
	}
	if !dirty && hash != "" {
		t.Fatal("clean but has working tree hash")
	}
}

func TestLog(t *testing.T) {
	root := repoRootOrSkip(t)
	entries, err := Log(root, []string{"README.md"}, 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one commit touching README.md")
	}
	for _, e := range entries {
		if e.Commit == "" {
			t.Fatal("empty commit hash")
		}
		if e.Author == "" {
			t.Fatal("empty author")
		}
	}
	t.Logf("got %d log entries, newest: %s by %s", len(entries), entries[0].Commit[:8], entries[0].Author)
}

func TestLineHistory(t *testing.T) {
	root := repoRootOrSkip(t)
	// Use a file known to exist
	entries, err := LineHistory(root, "README.md", 1, 5)
	if err != nil {
		t.Fatalf("LineHistory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one commit touching README.md lines 1-5")
	}
	t.Logf("line history: %d entries", len(entries))
}

func TestResolveRevision(t *testing.T) {
	root := repoRootOrSkip(t)
	hash, err := ResolveRevision(root, "HEAD")
	if err != nil {
		t.Fatalf("ResolveRevision: %v", err)
	}
	if len(hash) < 7 {
		t.Fatalf("resolved hash too short: %q", hash)
	}
}

func TestErrNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := RepoRoot(dir)
	if err != ErrNotAGitRepo {
		t.Fatalf("expected ErrNotAGitRepo, got: %v", err)
	}
}
