// Package html renders a compiled Lawbook to a human-readable HTML document.
// It operates on the Lawbook IR only, never on Markdown directly (PLAN1
// §22-§23).
package html

import (
	"errors"
	"io"

	"github.com/athreyac4/agentlaws/internal/model"
)

// ErrNotImplemented is returned until the HTML renderer is implemented per
// PLAN1 §64 Milestone 3.
var ErrNotImplemented = errors.New("renderer/html: not implemented")

// Render writes the HTML representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	return ErrNotImplemented
}
