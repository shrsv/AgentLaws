// Package pdf renders a compiled Lawbook to PDF, from the same Lawbook IR
// used by the HTML renderer (PLAN1 §23), not from Markdown directly.
//
// This renderer uses goldmark-pdf to leverage goldmark's full CommonMark +
// GFM parsing (headings, bold, italic, links, tables, code blocks, etc.)
// instead of a hand-rolled markdown subset.
package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"

	"github.com/shrsv/AgentLaws/internal/model"
)

// markdownPDF is the goldmark instance configured for PDF output.
var markdownPDF = goldmark.New(
	goldmark.WithRenderer(
		pdf.New(
			pdf.WithEscapeHTML(false),
		),
	),
)

// Render writes the PDF representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	md := buildMarkdown(book)
	return markdownPDF.Convert([]byte(md), w)
}

// RenderAll writes one combined PDF covering every book in books, each
// starting on a fresh page under title - the "export everything under
// this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	for i, book := range books {
		if i > 0 {
			b.WriteString("\n\\newpage\n\n")
		}
		buildMarkdownInto(&b, book)
	}
	return markdownPDF.Convert([]byte(b.String()), w)
}

// buildMarkdown converts a Lawbook IR into a Markdown string that
// goldmark-pdf can render with full CommonMark + GFM support.
func buildMarkdown(book model.Lawbook) string {
	var b strings.Builder
	buildMarkdownInto(&b, book)
	return b.String()
}

func buildMarkdownInto(b *strings.Builder, book model.Lawbook) {
	fmt.Fprintf(b, "# %s\n\n", book.Metadata.Title)

	for _, s := range book.Sections {
		// Section heading
		level := s.Level + 1
		if level > 6 {
			level = 6
		}
		fmt.Fprintf(b, "%s %s %s\n\n", strings.Repeat("#", level), s.Number, s.Title)

		// Section ID as a muted line
		fmt.Fprintf(b, "*%s*\n\n", s.ID)

		// Commentary (already markdown)
		if s.Commentary != "" {
			b.WriteString(s.Commentary)
			b.WriteString("\n\n")
		}

		// Laws
		for _, law := range s.Laws {
			fmt.Fprintf(b, "**%s** %s\n\n", law.Number, law.Text)
		}
	}

	// Provenance footer
	if book.Provenance.Revision != "" || book.Provenance.CompiledAt != "" || book.Provenance.AgentLawsVersion != "" {
		b.WriteString("---\n\n")
		var parts []string
		if book.Provenance.Revision != "" {
			short := book.Provenance.Revision
			if len(short) > 12 {
				short = short[:12]
			}
			dirty := ""
			if book.Provenance.Dirty {
				dirty = " (dirty)"
			}
			parts = append(parts, fmt.Sprintf("revision %s%s", short, dirty))
		}
		if book.Provenance.CompiledAt != "" {
			parts = append(parts, fmt.Sprintf("compiled %s", book.Provenance.CompiledAt))
		}
		if book.Provenance.CompilerName != "" {
			parts = append(parts, fmt.Sprintf("by %s", book.Provenance.CompilerName))
		}
		if book.Provenance.AgentLawsVersion != "" {
			v := book.Provenance.AgentLawsVersion
			if book.Provenance.AgentLawsBuildTime != "" {
				v += " (built " + book.Provenance.AgentLawsBuildTime + ")"
			}
			parts = append(parts, fmt.Sprintf("alaws %s", v))
		}
		fmt.Fprintf(b, "*%s*\n", strings.Join(parts, " · "))
	}
}
