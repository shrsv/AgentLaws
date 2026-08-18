package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeCluster(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alaws.toml"), []byte("title = \"test\""), 0o644); err != nil {
		t.Fatalf("write alaws.toml: %v", err)
	}
}

func TestFindClusters_Recursive(t *testing.T) {
	root := t.TempDir()
	writeCluster(t, filepath.Join(root, "books", "one"))
	writeCluster(t, filepath.Join(root, "books", "nested", "two"))

	clusters, err := FindClusters(root)
	if err != nil {
		t.Fatalf("FindClusters: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters, want 2: %+v", len(clusters), clusters)
	}
	for _, c := range clusters {
		if c.Title != "test" {
			t.Errorf("cluster %s: got title %q, want %q", c.Path, c.Title, "test")
		}
	}
}

func TestFindClusters_UnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are not enforced")
	}

	root := t.TempDir()
	writeCluster(t, filepath.Join(root, "books", "before"))
	writeCluster(t, filepath.Join(root, "books", "after"))
	writeCluster(t, filepath.Join(root, "locked"))

	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	clusters, err := FindClusters(root)
	if err != nil {
		t.Fatalf("FindClusters must not fail on an unreadable subdir, got: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters, want 2 (unreadable one skipped, scan continues): %+v", len(clusters), clusters)
	}
}

func TestFindClusters_UnreadableRoot(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("skipping: root or windows")
	}

	root := filepath.Join(t.TempDir(), "secret")
	if err := os.Mkdir(root, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	if _, err := FindClusters(root); err == nil {
		t.Fatal("expected error when the root itself is unreadable, got nil")
	}
}
