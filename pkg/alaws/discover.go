package alaws

import "github.com/shrsv/AgentLaws/internal/discovery"

// BookInfo identifies one discovered lawbook cluster.
type BookInfo struct {
	Path       string // directory containing alaws.toml
	ConfigPath string // path to alaws.toml itself
	Title      string // the book's title, from alaws.toml (PLAN1 §4)
}

// Discover finds every lawbook cluster (a directory containing alaws.toml)
// under root, skipping .git, node_modules, vendor, build, and dist. This is
// the single implementation behind `alaws books list`, the web API's
// GET /api/books, and context inference (docs/PLAN1.md §21, §56).
func Discover(root string) ([]BookInfo, error) {
	clusters, err := discovery.FindClusters(root)
	if err != nil {
		return nil, err
	}
	books := make([]BookInfo, len(clusters))
	for i, c := range clusters {
		books[i] = BookInfo{Path: c.Path, ConfigPath: c.ConfigPath, Title: c.Title}
	}
	return books, nil
}
