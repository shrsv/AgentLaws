// Package discovery finds lawbook clusters (directories containing
// alaws.toml) and detects Markdown files present on disk but absent from a
// cluster's ordering. See docs/PLAN1.md §21, §56, §34.
package discovery

import "errors"

// ErrNotImplemented is returned by every stub in this package until
// discovery is implemented per PLAN1 §64 Milestone 2.
var ErrNotImplemented = errors.New("discovery: not implemented")

// Cluster is a discovered lawbook cluster.
type Cluster struct {
	Path       string // directory containing alaws.toml
	ConfigPath string // path to alaws.toml itself
}

// FindClusters recursively searches root for alaws.toml files, skipping
// .git, node_modules, vendor, build, and dist.
func FindClusters(root string) ([]Cluster, error) {
	return nil, ErrNotImplemented
}

// UnorderedFiles returns Markdown files under a cluster's directory that are
// not referenced by its ordering (PLAN1 §21) - diagnostic only.
func UnorderedFiles(cluster Cluster, ordering []string) ([]string, error) {
	return nil, ErrNotImplemented
}
