// Package discovery finds lawbook clusters (directories containing
// alaws.toml) and detects Markdown files present on disk but absent from a
// cluster's ordering. See docs/PLAN1.md §21, §56.
package discovery

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/shrsv/AgentLaws/internal/parser"
)

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
}

// Cluster is a discovered lawbook cluster.
type Cluster struct {
	Path       string // directory containing alaws.toml
	ConfigPath string // path to alaws.toml itself
	Title      string // the book's title, from alaws.toml - used for organizing (PLAN1 §4)
}

// FindClusters recursively searches root for alaws.toml files, skipping
// .git, node_modules, vendor, build, and dist. Each cluster's Title is
// read from its alaws.toml on a best-effort basis: a malformed config
// still produces a Cluster (with an empty Title) so a whole-repo scan
// isn't derailed by one broken book - `alaws validate` is the tool for
// surfacing that book's problems specifically.
func FindClusters(root string) ([]Cluster, error) {
	var clusters []Cluster
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "alaws.toml" {
			c := Cluster{Path: filepath.Dir(path), ConfigPath: path}
			if meta, err := parser.ParseLawbookConfig(path); err == nil {
				c.Title = meta.Title
			}
			clusters = append(clusters, c)
		}
		return nil
	})
	return clusters, err
}

// UnorderedFiles returns Markdown files under a cluster's directory that are
// not referenced by its ordering (PLAN1 §21) - diagnostic only.
func UnorderedFiles(cluster Cluster, ordering []string) ([]string, error) {
	ordered := make(map[string]bool, len(ordering))
	for _, o := range ordering {
		ordered[filepath.ToSlash(o)] = true
	}

	var unordered []string
	err := filepath.WalkDir(cluster.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != cluster.Path && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".md" && ext != ".mdx" {
			return nil
		}
		rel, err := filepath.Rel(cluster.Path, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !ordered[rel] {
			unordered = append(unordered, rel)
		}
		return nil
	})
	return unordered, err
}
