// Package pdf renders a compiled Lawbook to PDF, from the same Lawbook IR
// used by the HTML renderer (PLAN1 §23), not from Markdown directly.
package pdf

import (
	"io"

	"github.com/go-pdf/fpdf"

	"github.com/athreyac4/agentlaws/internal/model"
)

// Render writes the PDF representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(20, 20, 20)
	doc.SetAutoPageBreak(true, 20)
	doc.AddPage()

	doc.SetFont("Helvetica", "B", 20)
	doc.MultiCell(0, 10, book.Metadata.Title, "", "L", false)
	doc.Ln(4)

	for _, s := range book.Sections {
		doc.SetFont("Helvetica", "B", headingSize(s.Level))
		doc.MultiCell(0, 8, s.Number+"  "+s.Title, "", "L", false)

		doc.SetFont("Helvetica", "I", 9)
		doc.SetTextColor(120, 120, 120)
		doc.MultiCell(0, 5, s.ID, "", "L", false)
		doc.SetTextColor(0, 0, 0)

		if s.Commentary != "" {
			doc.SetFont("Helvetica", "", 11)
			doc.MultiCell(0, 6, s.Commentary, "", "L", false)
			doc.Ln(2)
		}

		for _, law := range s.Laws {
			doc.SetFont("Helvetica", "B", 11)
			doc.CellFormat(14, 6, law.Number, "", 0, "L", false, 0, "")
			doc.SetFont("Helvetica", "", 11)
			doc.MultiCell(0, 6, law.Text, "", "L", false)
		}
		doc.Ln(4)
	}

	return doc.Output(w)
}

func headingSize(level int) float64 {
	switch level {
	case 1:
		return 16
	case 2:
		return 13
	default:
		return 11
	}
}
