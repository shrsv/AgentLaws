package alaws

import (
	"fmt"
	"path/filepath"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/ordering"
	"github.com/athreyac4/agentlaws/internal/parser"
)

// ConfigPath resolves a book argument (a directory, or an explicit
// alaws.toml path) to the path of its alaws.toml. The CLI, the web API,
// and any other pkg/alaws caller all use this one resolution rule.
func ConfigPath(book string) string {
	return compiler.ConfigPath(book)
}

// CreateBook creates a new alaws.toml at path with the given title and an
// empty ordering, establishing a new lawbook cluster (PLAN1 §4).
func CreateBook(path, title string) error {
	return ordering.NewBook(path, title)
}

// Title returns book's title, from its alaws.toml, without compiling its
// sections - used by `alaws books show` and anywhere else a caller wants a
// book's name without paying for a full compile.
func Title(book string) (string, error) {
	meta, err := parser.ParseLawbookConfig(ConfigPath(book))
	if err != nil {
		return "", err
	}
	return meta.Title, nil
}

// Node is one chapter or section in a book's ordering tree, with its
// derived presentation Level and ParentID (docs/PLAN1.md §32).
type Node struct {
	Path     string
	ID       string
	Level    int
	ParentID string // "" for a top-level chapter
}

func nodeFrom(n ordering.Node) Node {
	return Node{Path: n.Path, ID: n.ID, Level: n.Level, ParentID: n.ParentID}
}

// Tree returns book's chapters and sections in ordering order, with each
// entry's derived presentation level and parent (docs/PLAN1.md §8, §32).
func Tree(book string) ([]Node, error) {
	nodes, err := ordering.Tree(ConfigPath(book))
	if err != nil {
		return nil, err
	}
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = nodeFrom(n)
	}
	return out, nil
}

// Placement describes where a new or moved chapter/section goes relative
// to the existing ordering, checked in this order: Position (1-based
// absolute), then Before (immediately ahead of that entry's subtree), then
// After (immediately after that entry's entire subtree), then append at
// the end. See internal/ordering.Placement (docs/PLAN1.md §32).
type Placement struct {
	After    string
	Before   string
	Position int
}

func (p Placement) toOrdering() ordering.Placement {
	return ordering.Placement{After: p.After, Before: p.Before, Position: p.Position}
}

// levelOverride decides whether a newly created or moved section needs an
// explicit `level:` written into its frontmatter: only when desired
// diverges from what the file's own folder depth would produce by default
// (docs/PLAN1.md §8). Returns 0 (meaning "omit it") when they already
// agree - the common case.
func levelOverride(entryPath string, desired int) int {
	if parser.ResolveLevel(entryPath, nil) == desired {
		return 0
	}
	return desired
}

// CreateChapter creates a new top-level section (a "chapter") in book at
// file, and inserts it into the ordering per placement (the zero value
// appends at the end). An explicit level override is written to the file
// only if file's own folder depth wouldn't already default to level 1.
func CreateChapter(book, file, title, id string, placement Placement) error {
	path := filepath.Join(book, file)
	meta := ordering.SectionMeta{Title: title, ID: id, Level: levelOverride(file, 1)}
	if err := ordering.NewSectionFile(path, meta); err != nil {
		return err
	}
	return ordering.Insert(ConfigPath(book), file, placement.toOrdering())
}

// CreateSection creates a new nested section in book at file, under
// parentID, and inserts it into the ordering per placement (the zero
// value appends as parentID's last child). The desired level is
// parentID's level + 1 unless level > 0 overrides it; an explicit
// override is written to the file only if its folder depth wouldn't
// already produce that level.
func CreateSection(book, file, title, id, parentID string, level int, placement Placement) error {
	path := filepath.Join(book, file)

	explicitLevel := level > 0
	resolvedLevel := level
	if !explicitLevel {
		nodes, err := ordering.Tree(ConfigPath(book))
		if err != nil {
			return err
		}
		found := false
		for _, n := range nodes {
			if n.ID == parentID {
				resolvedLevel = n.Level + 1
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parent %q not found in %s", parentID, book)
		}
	}

	metaLevel := resolvedLevel
	if !explicitLevel {
		metaLevel = levelOverride(file, resolvedLevel)
	}
	if err := ordering.NewSectionFile(path, ordering.SectionMeta{Title: title, ID: id, Level: metaLevel}); err != nil {
		return err
	}

	p := placement.toOrdering()
	if p.After == "" && p.Before == "" && p.Position == 0 {
		p.After = parentID
	}
	return ordering.Insert(ConfigPath(book), file, p)
}

// MoveChapter relocates a chapter, along with its section subtree, to a
// new position.
func MoveChapter(book, id string, placement Placement) error {
	return ordering.Move(ConfigPath(book), id, placement.toOrdering())
}

// MoveSection relocates a section to a new position and, if newParentID is
// non-empty, under a new parent - fixing up the section file's frontmatter
// level to match the new parent regardless of where the file physically
// lives (the move never relocates the file itself).
func MoveSection(book, id, newParentID string, placement Placement) error {
	p := placement.toOrdering()
	if newParentID != "" && p.After == "" && p.Before == "" && p.Position == 0 {
		p.After = newParentID
	}
	if err := ordering.Move(ConfigPath(book), id, p); err != nil {
		return err
	}
	if newParentID == "" {
		return nil
	}

	nodes, err := ordering.Tree(ConfigPath(book))
	if err != nil {
		return err
	}
	var childPath string
	parentLevel := -1
	for _, n := range nodes {
		if n.ID == id {
			childPath = n.Path
		}
		if n.ID == newParentID {
			parentLevel = n.Level
		}
	}
	if childPath == "" {
		return fmt.Errorf("section %q not found after move", id)
	}
	if parentLevel == -1 {
		return fmt.Errorf("parent %q not found", newParentID)
	}
	return ordering.SetLevel(filepath.Join(book, childPath), levelOverride(childPath, parentLevel+1))
}

// RemoveChapter removes a chapter (and, if force, its section subtree)
// from book's ordering. The underlying file(s) are left on disk, excluded
// from the lawbook rather than deleted (PLAN1 §21).
func RemoveChapter(book, id string, force bool) error {
	return ordering.Remove(ConfigPath(book), id, force)
}

// RemoveSection removes a section from book's ordering. The underlying
// file is left on disk.
func RemoveSection(book, id string, force bool) error {
	return ordering.Remove(ConfigPath(book), id, force)
}
