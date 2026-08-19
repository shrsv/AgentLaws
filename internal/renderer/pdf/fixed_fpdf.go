package pdf

import (
	"io"

	pdflib "github.com/stephenafamo/goldmark-pdf"
)

// internalLinkPos is the position and size of one piece of link text
// written by WriteInternalLink, recorded so a proper /Dest annotation can
// be placed over it once every anchor in the document is known.
type internalLinkPos struct {
	page          int
	x, y          float64
	width, height float64
	anchor        string
}

// fixedFpdf wraps goldmark-pdf's own Fpdf adapter (pdflib.Fpdf) to fix a
// bug in its internal-link handling, confirmed by experiment before this
// code was written: Fpdf.WriteInternalLink writes an *immediate* link
// annotation via gofpdf's WriteLinkString("#"+anchor) - but "#anchor" is
// not a valid absolute URI, so PDF viewers treat it as a broken/insecure
// external link (this is what Adobe's "open link" prompt is reacting to,
// and why Chrome/SumatraPDF don't navigate at all). Separately, at
// Write() time, Fpdf *also* places a second, correct /Dest annotation on
// the exact same rectangle. The result is two stacked annotations per
// link, and which one a viewer honors is inconsistent.
//
// pdflib.Fpdf's own bookkeeping for the (correct) deferred half lives in
// unexported fields (anchorLinks, anchors), so it can't be reused
// piecemeal from outside the package. This type reimplements just the
// internal-link path using only pdflib.Fpdf's exported surface (its
// embedded *gofpdf.Fpdf, reached via the Fpdf field) and never calls
// WriteLinkString: WriteInternalLink writes plain text and records its
// position; AddInternalLink registers the target the same way the
// library does; Write() resolves pending positions against registered
// anchors into a single Link() annotation each, then delegates to
// gofpdf's Output.
//
// Every other PDF interface method is unchanged, inherited by embedding
// *pdflib.Fpdf. Construct fresh per render (see newMarkdownPDF) - it
// carries per-document state that must not leak between renders.
type fixedFpdf struct {
	*pdflib.Fpdf
	anchors map[string]int
	pending []internalLinkPos
}

func newFixedFpdf(inner *pdflib.Fpdf) *fixedFpdf {
	return &fixedFpdf{Fpdf: inner, anchors: map[string]int{}}
}

func (f *fixedFpdf) AddInternalLink(anchor string) {
	raw := f.Fpdf.Fpdf
	id := raw.AddLink()
	raw.SetLink(id, raw.GetY(), -1)
	f.anchors[anchor] = id
}

func (f *fixedFpdf) WriteInternalLink(lineHeight float64, text string, anchor string) {
	raw := f.Fpdf.Fpdf
	f.pending = append(f.pending, internalLinkPos{
		page:   raw.PageNo(),
		x:      raw.GetX(),
		y:      raw.GetY(),
		width:  raw.GetStringWidth(text),
		height: lineHeight,
		anchor: anchor,
	})
	raw.Write(lineHeight, text)
}

func (f *fixedFpdf) Write(w io.Writer) error {
	raw := f.Fpdf.Fpdf
	for _, link := range f.pending {
		id, ok := f.anchors[link.anchor]
		if !ok {
			// Dangling alaws: reference - resolver-level validation
			// (the dangling-reference diagnostic) is what should catch
			// this before export; skip rather than panic so a bad
			// reference degrades to inert text instead of a broken PDF.
			continue
		}
		raw.SetPage(link.page)
		raw.Link(link.x, link.y, link.width, link.height, id)
	}
	raw.SetPage(raw.PageCount())
	return raw.Output(w)
}
