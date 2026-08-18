// Package numbering assigns canonical presentation numbers (e.g. "2.5.3") to
// sections and laws based on lawbook ordering. See docs/PLAN1.md §10, §34.
package numbering

import (
	"errors"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned by every stub in this package until
// numbering is implemented per PLAN1 §64 Milestone 2.
var ErrNotImplemented = errors.New("numbering: not implemented")

// Assign computes Section.Number and Law.Number for every section, based on
// each section's Level and position in the ordered slice. It also derives
// each section's ParentID per the outline rule in PLAN1 §32 (nearest
// preceding section with a lower Level).
func Assign(sections []model.Section) ([]model.Section, error) {
	return nil, ErrNotImplemented
}
