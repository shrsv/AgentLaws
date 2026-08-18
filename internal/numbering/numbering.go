// Package numbering assigns canonical presentation numbers (e.g. "2.5.3") to
// sections and laws based on lawbook ordering. See docs/PLAN1.md §10.
package numbering

import (
	"fmt"

	"github.com/athreyac4/agentlaws/internal/model"
)

// Assign computes Section.Number and Law.Number for every section, based on
// each section's Level and position in the ordered slice. It also derives
// each section's ParentID: the nearest preceding section with a lower
// Level, per the outline rule in docs/PLAN1.md §32 ("chapter"/"section"
// vocabulary). A section with no such predecessor is top-level
// (ParentID == "").
//
// Numbering is outline-style and works at any depth: a top-level section
// gets a sequential number ("1", "2", ...); a nested section gets its
// parent's number plus a sequential index among that parent's own children
// ("2.1", "2.2", ...). A law's number is its section's number plus its
// 1-based position within the section ("2.5.3").
func Assign(sections []model.Section) ([]model.Section, error) {
	out := make([]model.Section, len(sections))
	copy(out, sections)

	levels := make([]int, len(out))
	for i, s := range out {
		levels[i] = s.Level
	}

	childCount := map[int]int{} // parent index (-1 = top-level) -> children seen so far

	for i := range out {
		parent := parentIndex(levels, i)
		childCount[parent]++
		if parent == -1 {
			out[i].Number = fmt.Sprintf("%d", childCount[parent])
			out[i].ParentID = ""
		} else {
			out[i].Number = fmt.Sprintf("%s.%d", out[parent].Number, childCount[parent])
			out[i].ParentID = out[parent].ID
		}
	}

	for i := range out {
		for j := range out[i].Laws {
			out[i].Laws[j].Number = fmt.Sprintf("%s.%d", out[i].Number, j+1)
			out[i].Laws[j].Index = j + 1
			out[i].Laws[j].SectionID = out[i].ID
		}
	}

	return out, nil
}

// parentIndex returns the index of the nearest preceding entry in levels
// with a lower level than levels[i], or -1 if there is none.
func parentIndex(levels []int, i int) int {
	for j := i - 1; j >= 0; j-- {
		if levels[j] < levels[i] {
			return j
		}
	}
	return -1
}
