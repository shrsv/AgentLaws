package alaws

import (
	"fmt"
	"path/filepath"

	"github.com/shrsv/AgentLaws/internal/lawedit"
	"github.com/shrsv/AgentLaws/internal/ordering"
	"github.com/shrsv/AgentLaws/internal/parser"
	"github.com/shrsv/AgentLaws/internal/resolver"
)

// SectionFilePath resolves a section ID to its source file path by walking
// book's ordering tree.
func SectionFilePath(book, id string) (string, error) {
	nodes, err := ordering.Tree(ConfigPath(book))
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.ID == id {
			return filepath.Join(book, n.Path), nil
		}
	}
	return "", fmt.Errorf("%w: section %q", resolver.ErrNotFound, id)
}

// AddLaw appends a new numbered clause to sectionID's laws region. If
// after > 0, the clause is inserted immediately after that existing
// clause number instead of at the end.
func AddLaw(book, sectionID, text string, after int) error {
	path, err := SectionFilePath(book, sectionID)
	if err != nil {
		return err
	}
	return lawedit.Add(path, text, after)
}

// RemoveLaw deletes the numbered clause `number` from sectionID's laws
// region, renumbering subsequent clauses.
func RemoveLaw(book, sectionID string, number int, force bool) error {
	path, err := SectionFilePath(book, sectionID)
	if err != nil {
		return err
	}
	return lawedit.Remove(path, number, force)
}

// ListLaws returns sectionID's numbered clauses as authored (before
// canonical numbering is assigned at compile time).
func ListLaws(book, sectionID string) ([]string, error) {
	path, err := SectionFilePath(book, sectionID)
	if err != nil {
		return nil, err
	}
	parsed, err := parser.ParseSection(path)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(parsed.RawLaws))
	for i, l := range parsed.RawLaws {
		out[i] = l.Text
	}
	return out, nil
}
