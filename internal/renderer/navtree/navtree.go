// Package navtree builds the same section hierarchy the alaws web UI's
// sidebar shows (web/src/views/BookDetail.tsx's buildTree), so every
// static exporter can drive its navigation/outline straight from the
// compiled Lawbook IR instead of scanning rendered Markdown/HTML for
// heading tags - which would also pick up headings authors write inside
// section commentary or prompt template bodies for prose formatting.
package navtree

import "github.com/shrsv/AgentLaws/internal/model"

// Node is a Section plus its child sections, nested by Section.Level.
type Node struct {
	Section  model.Section
	Children []*Node
}

// Build rebuilds sections - a flat, depth-first list ordered by
// Level/ParentID - into a nested tree, mirroring buildTree in
// BookDetail.tsx exactly: each node's children are the sections that
// follow it, up until a section at the same or shallower Level appears.
func Build(sections []model.Section) []*Node {
	var roots []*Node
	var stack []*Node
	for _, s := range sections {
		node := &Node{Section: s}
		for len(stack) > 0 && stack[len(stack)-1].Section.Level >= s.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
		stack = append(stack, node)
	}
	return roots
}
