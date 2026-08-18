package pdf

import (
	"regexp"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Commentary and law text are Markdown (docs/PLAN1.md §7). fpdf has no
// Markdown support of its own, so this file is a small, deliberately
// narrow renderer covering only what this project's lawbooks actually
// use: paragraphs, "* "/"- " bullet lists, backtick-delimited inline
// code, and fenced ``` code blocks. It does not attempt full CommonMark -
// headings, links, emphasis, and nested lists inside law/commentary text
// are out of scope; the HTML renderer (goldmark) is the complete
// implementation these formats are compiled from the same source as.
//
// Every string passed to fpdf must go through tr (pdf.go's
// UnicodeTranslatorFromDescriptor) before being written - fpdf's core
// fonts use the cp1252 code page, not UTF-8, so untranslated non-ASCII
// text (including the "•" this file writes for list bullets) would
// otherwise render as mojibake.

var codeSpanRe = regexp.MustCompile("`([^`]+)`")

const bodyLineHeight = 6.0

// writeInlineRuns writes a single paragraph of text (no block-level
// constructs), switching to a monospace font for backtick-delimited code
// spans, without breaking to a new line first or after.
func writeInlineRuns(doc *fpdf.Fpdf, tr func(string) string, text string, size float64) {
	idx := 0
	for _, loc := range codeSpanRe.FindAllStringSubmatchIndex(text, -1) {
		if loc[0] > idx {
			doc.SetFont("Helvetica", "", size)
			doc.Write(bodyLineHeight, tr(text[idx:loc[0]]))
		}
		doc.SetFont("Courier", "", size)
		doc.Write(bodyLineHeight, tr(text[loc[2]:loc[3]]))
		idx = loc[1]
	}
	if idx < len(text) {
		doc.SetFont("Helvetica", "", size)
		doc.Write(bodyLineHeight, tr(text[idx:]))
	}
}

// writeMarkdownBlock renders a full Markdown snippet - paragraphs, bullet
// lists, and fenced code blocks - such as a section's commentary.
func writeMarkdownBlock(doc *fpdf.Fpdf, tr func(string) string, md string, size float64) {
	lines := strings.Split(md, "\n")
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		switch {
		case strings.HasPrefix(trimmed, "```"):
			i++
			start := i
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				i++
			}
			writeCodeBlock(doc, tr, lines[start:i], size)
			if i < len(lines) {
				i++ // skip closing fence
			}

		case trimmed == "":
			i++

		case strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- "):
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(t, "* ") && !strings.HasPrefix(t, "- ") {
					break
				}
				doc.SetFont("Helvetica", "", size)
				doc.Write(bodyLineHeight, tr("  •  "))
				writeInlineRuns(doc, tr, t[2:], size)
				doc.Ln(-1)
				i++
			}
			doc.Ln(2)

		default:
			var para []string
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if t == "" || strings.HasPrefix(t, "```") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "- ") {
					break
				}
				para = append(para, t)
				i++
			}
			writeInlineRuns(doc, tr, strings.Join(para, " "), size)
			doc.Ln(-1)
			doc.Ln(2)
		}
	}
}

// writeCodeBlock renders a fenced code block as a shaded, monospace block
// - the "should be in a code block" treatment, without syntax highlighting
// (unlike the HTML renderer, which uses goldmark-highlighting/chroma;
// per-token color runs are impractical in fpdf's cell model for the
// return this project needs).
func writeCodeBlock(doc *fpdf.Fpdf, tr func(string) string, lines []string, size float64) {
	doc.SetFont("Courier", "", size-1)
	doc.SetFillColor(245, 245, 245)
	doc.SetTextColor(40, 40, 40)
	for _, l := range lines {
		doc.CellFormat(0, 5, tr("  "+l), "", 1, "L", true, 0, "")
	}
	doc.SetTextColor(0, 0, 0)
	doc.Ln(2)
}
