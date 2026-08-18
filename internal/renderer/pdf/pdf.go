// Package pdf renders a compiled Lawbook to PDF, from the same Lawbook IR
// used by the HTML renderer (PLAN1 §23), not from Markdown directly.
package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/shrsv/AgentLaws/internal/model"
)

// newDoc creates and configures the shared fpdf.Fpdf setup, plus the
// Unicode translator every text-writing call in this package must pass its
// strings through. fpdf's core fonts (Helvetica, Courier) use the cp1252
// code page, not raw UTF-8; without translation, any non-ASCII character
// (an em dash, a curly quote, "§", the bullet this package uses for lists)
// renders as mojibake instead of the character itself.
func newDoc() (*fpdf.Fpdf, func(string) string) {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(20, 20, 20)
	doc.SetAutoPageBreak(true, 20)
	return doc, doc.UnicodeTranslatorFromDescriptor("")
}

// Render writes the PDF representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	doc, tr := newDoc()
	doc.AddPage()

	doc.SetFont("Helvetica", "B", 20)
	doc.MultiCell(0, 10, tr(book.Metadata.Title), "", "L", false)
	doc.Ln(4)

	renderSections(doc, tr, book.Sections)
	renderProvenanceFooter(doc, tr, book.Provenance)
	return doc.Output(w)
}

// RenderAll writes one combined PDF covering every book in books, each
// starting on a fresh page under title - the "export everything under
// this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	doc, tr := newDoc()
	doc.AddPage()

	doc.SetFont("Helvetica", "B", 24)
	doc.MultiCell(0, 12, tr(title), "", "L", false)
	doc.Ln(6)

	for _, book := range books {
		doc.AddPage()
		doc.SetFont("Helvetica", "B", 20)
		doc.MultiCell(0, 10, tr(book.Metadata.Title), "", "L", false)
		doc.Ln(4)
		renderSections(doc, tr, book.Sections)
		renderProvenanceFooter(doc, tr, book.Provenance)
	}

	return doc.Output(w)
}

func renderSections(doc *fpdf.Fpdf, tr func(string) string, sections []model.Section) {
	for _, s := range sections {
		doc.SetFont("Helvetica", "B", headingSize(s.Level))
		doc.MultiCell(0, 8, tr(s.Number+"  "+s.Title), "", "L", false)

		doc.SetFont("Helvetica", "I", 9)
		doc.SetTextColor(120, 120, 120)
		doc.MultiCell(0, 5, tr(s.ID), "", "L", false)
		doc.SetTextColor(0, 0, 0)

		if s.Commentary != "" {
			writeMarkdownBlock(doc, tr, s.Commentary, 11)
		}

		for _, law := range s.Laws {
			doc.SetFont("Helvetica", "B", 11)
			doc.CellFormat(14, 6, tr(law.Number), "", 0, "L", false, 0, "")
			writeInlineRuns(doc, tr, law.Text, 11)
			doc.Ln(-1)
		}
		doc.Ln(4)
	}
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

// renderProvenanceFooter writes provenance metadata at the bottom of a
// compiled PDF (docs/PLAN1.md §24-§25, §50). Silently omitted when
// Provenance is empty.
func renderProvenanceFooter(doc *fpdf.Fpdf, tr func(string) string, prov model.Provenance) {
	if prov.Revision == "" && prov.CompiledAt == "" && prov.AgentLawsVersion == "" {
		return
	}
	doc.Ln(6)
	doc.SetDrawColor(200, 200, 200)
	doc.Line(doc.GetX(), doc.GetY(), doc.GetX()+170, doc.GetY())
	doc.Ln(3)
	doc.SetFont("Helvetica", "", 8)
	doc.SetTextColor(120, 120, 120)

	var parts []string
	if prov.Revision != "" {
		short := prov.Revision
		if len(short) > 12 {
			short = short[:12]
		}
		dirty := ""
		if prov.Dirty {
			dirty = " (dirty)"
		}
		parts = append(parts, fmt.Sprintf("revision %s%s", short, dirty))
	}
	if prov.CompiledAt != "" {
		parts = append(parts, fmt.Sprintf("compiled %s", prov.CompiledAt))
	}
	if prov.CompilerName != "" {
		parts = append(parts, fmt.Sprintf("by %s", prov.CompilerName))
	}
	if prov.AgentLawsVersion != "" {
		v := prov.AgentLawsVersion
		if prov.AgentLawsBuildTime != "" {
			v += " (built " + prov.AgentLawsBuildTime + ")"
		}
		parts = append(parts, fmt.Sprintf("alaws %s", v))
	}
	doc.MultiCell(0, 4, tr(strings.Join(parts, " · ")), "", "L", false)
	doc.SetTextColor(0, 0, 0)
}
