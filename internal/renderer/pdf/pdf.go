// Package pdf renders a compiled Lawbook to PDF, from the same Lawbook IR
// used by the HTML renderer (PLAN1 §23), not from Markdown directly.
package pdf

import (
	"errors"
	"io"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned until the PDF renderer is implemented per
// PLAN1 §64 Milestone 10.
var ErrNotImplemented = errors.New("renderer/pdf: not implemented")

// Render writes the PDF representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	return ErrNotImplemented
}
