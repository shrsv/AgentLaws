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
	"regexp"
	"strings"

	pdflib "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/util"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
)

// markdownPDF is the goldmark instance configured for PDF output.
var markdownPDF = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithRenderer(
		pdflib.New(
			pdflib.WithEscapeHTML(false),
			pdflib.WithNodeRenderers(
				util.Prioritized(&anchorNodeRenderer{}, 100),
			),
		),
	),
)

// anchorSentinelRe matches <!--alaws-anchor:...--> sentinel comments.
var anchorSentinelRe = regexp.MustCompile(`^<!--alaws-anchor:(.+)-->$`)

// anchorNodeRenderer handles raw HTML nodes that are alaws-anchor
// sentinels, registering them as internal PDF link anchors.
type anchorNodeRenderer struct{}

func (r *anchorNodeRenderer) RegisterFuncs(reg pdflib.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
}

func (r *anchorNodeRenderer) renderRawHTML(w *pdflib.Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.RawHTML)
	segs := n.Segments
	var content string
	for i := 0; i < segs.Len(); i++ {
		seg := segs.At(i)
		content += string(seg.Value(source))
	}
	if m := anchorSentinelRe.FindStringSubmatch(content); m != nil {
		w.Pdf.AddInternalLink(m[1])
	}
	return ast.WalkContinue, nil
}

// ResolveFunc resolves an alaws: link token to an href.
type ResolveFunc func(token string) (href string, ok bool)

// makeResolveFunc builds a ResolveFunc for a single book.
func makeResolveFunc(book model.Lawbook) ResolveFunc {
	return func(token string) (string, bool) {
		r, err := resolver.Resolve(book, token)
		if err != nil {
			return "", false
		}
		return "#" + resolver.AnchorFor(r), true
	}
}

// makeCombinedResolveFunc builds a ResolveFunc that searches all books.
func makeCombinedResolveFunc(books []model.Lawbook) ResolveFunc {
	return func(token string) (string, bool) {
		for _, book := range books {
			r, err := resolver.Resolve(book, token)
			if err != nil {
				continue
			}
			return "#" + resolver.AnchorFor(r), true
		}
		return "", false
	}
}

// Render writes the PDF representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	resolve := makeResolveFunc(book)
	md := buildMarkdown(book, resolve)
	return markdownPDF.Convert([]byte(md), w)
}

// RenderAll writes one combined PDF covering every book in books, each
// starting on a fresh page under title - the "export everything under
// this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	resolve := makeCombinedResolveFunc(books)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	for i, book := range books {
		if i > 0 {
			b.WriteString("\n\\newpage\n\n")
		}
		buildMarkdownInto(&b, book, resolve)
	}
	return markdownPDF.Convert([]byte(b.String()), w)
}

// buildMarkdown converts a Lawbook IR into a Markdown string that
// goldmark-pdf can render with full CommonMark + GFM support.
func buildMarkdown(book model.Lawbook, resolve ResolveFunc) string {
	var b strings.Builder
	buildMarkdownInto(&b, book, resolve)
	return b.String()
}

func buildMarkdownInto(b *strings.Builder, book model.Lawbook, resolve ResolveFunc) {
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

		// Commentary (already markdown — rewrite alaws: links)
		if s.Commentary != "" {
			b.WriteString(rewriteAlawsLinks(s.Commentary, resolve))
			b.WriteString("\n\n")
		}

		// Laws
		for _, law := range s.Laws {
			anchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
			fmt.Fprintf(b, "<!--alaws-anchor:%s-->\n", anchor)
			fmt.Fprintf(b, "**%s** %s\n\n", law.Number, rewriteAlawsLinks(law.Text, resolve))
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

// alawsLinkRe matches Markdown links with alaws: destinations.
var alawsLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(alaws:([^)]+)\)`)

// rewriteAlawsLinks replaces [text](alaws:token) links with
// [text](#anchor) links using the resolver.
func rewriteAlawsLinks(md string, resolve ResolveFunc) string {
	if resolve == nil {
		return md
	}
	return alawsLinkRe.ReplaceAllStringFunc(md, func(match string) string {
		m := alawsLinkRe.FindStringSubmatch(match)
		if m == nil {
			return match
		}
		text, token := m[1], m[2]
		href, ok := resolve(token)
		if !ok {
			return match
		}
		return fmt.Sprintf("[%s](%s)", text, href)
	})
}
