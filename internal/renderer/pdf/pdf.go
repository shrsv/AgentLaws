// Package pdf renders a compiled Lawbook to PDF, from the same Lawbook IR
// used by the HTML renderer (PLAN1 §23), not from Markdown directly.
//
// This renderer uses goldmark-pdf to leverage goldmark's full CommonMark +
// GFM parsing (headings, bold, italic, links, tables, code blocks, etc.)
// instead of a hand-rolled markdown subset.
//
// goldmark-pdf@v0.4.2 has two bugs that matter for cross-referencing (see
// fixed_fpdf.go and textNodeRenderer below for the full diagnosis and
// fix of each, both confirmed by direct experiment against the vendored
// library before this code was written):
//
//   - Internal links (`[text](#anchor)`) get two overlapping annotations
//   - one broken, one correct - so viewers behave inconsistently.
//     Fixed by fixedFpdf.
//   - A paragraph that soft-wraps across source lines loses the space at
//     the wrap point, e.g. "...complemented byreview requirementsand
//     testing...". Fixed by textNodeRenderer.
//
// A third bug, not in the library itself but in this package's prior use of
// it, was found afterward: buildMarkdownInto only ever emitted an
// alaws-anchor sentinel before a law, never before a section heading, so an
// alaws: link whose target was a section rather than a law silently
// resolved to nothing (fixedFpdf.Write drops a pending link with no
// matching registered anchor rather than erroring). Fixed by emitting a
// sentinel before every section heading too.
package pdf

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"regexp"
	"strconv"
	"strings"

	pdflib "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/util"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
)

// LinkBlue is the standard link color for PDF links.
var LinkBlue = color.RGBA{R: 0x1A, G: 0x73, B: 0xE8, A: 0xFF}

// newMarkdownPDF builds a fresh goldmark instance configured for PDF
// output. It is constructed fresh on every call rather than shared as a
// package-level singleton: the fixedFpdf it wires in carries per-document
// state (registered anchors, pending internal-link positions) that must
// not leak between unrelated renders.
func newMarkdownPDF(bookmarks []bookmarkEntry) goldmark.Markdown {
	fpdf := pdflib.NewFpdf(context.Background(), pdflib.FpdfConfig{}, nil)
	fixed := newFixedFpdf(fpdf)

	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
		goldmark.WithRenderer(
			pdflib.New(
				pdflib.WithPDF(fixed),
				pdflib.WithEscapeHTML(false),
				pdflib.WithLinkColor(LinkBlue),
				pdflib.WithNodeRenderers(
					util.Prioritized(&anchorNodeRenderer{bookmarks: bookmarks}, 100),
					util.Prioritized(&textNodeRenderer{}, 100),
				),
			),
		),
	)
}

// anchorSentinelRe matches a <!--alaws-anchor:...--> sentinel comment,
// which buildMarkdownInto emits on its own line immediately before an
// anchorable law or section so this renderer can register it as an
// internal PDF link target.
var anchorSentinelRe = regexp.MustCompile(`<!--alaws-anchor:(.+?)-->`)

// bookmarkSentinelRe matches a <!--alaws-bookmark:N--> sentinel comment,
// which buildMarkdownInto emits immediately before a book title, group
// header, section heading, or prompt heading - never before a law or
// inside commentary/prompt-template Markdown - so this renderer can
// register exactly that structural node, and nothing else, as a
// /Outlines sidebar entry (via Pdf.Bookmark). N indexes into the
// bookmarks slice built alongside the Markdown by buildMarkdownInto.
var bookmarkSentinelRe = regexp.MustCompile(`<!--alaws-bookmark:(\d+)-->`)

// bookmarkEntry is one entry to register in the PDF's /Outlines sidebar:
// its display text and nesting level (0 = top).
type bookmarkEntry struct {
	Text  string
	Level int
}

// anchorNodeRenderer registers alaws-anchor sentinels as internal PDF
// link targets (via Pdf.AddInternalLink), so a later WriteInternalLink
// to that anchor (see fixedFpdf) has somewhere to resolve to, and
// alaws-bookmark sentinels as /Outlines sidebar entries (via
// Pdf.Bookmark). Both sentinel kinds are handled by this one renderer,
// rather than two separate ones, because goldmark-pdf's Register keeps
// only the last-registered func per ast.NodeKind (renderer.go) - a
// second renderer registered for the same KindHTMLBlock/KindRawHTML
// would silently replace this one instead of running alongside it.
//
// A standalone-line HTML comment parses as a block-level ast.KindHTMLBlock
// node, not inline ast.KindRawHTML - goldmark-pdf's default renderer for
// KindHTMLBlock is a no-op ("Cannot process HTML blocks"), so without
// this override, every sentinel was silently dropped. KindRawHTML is
// also handled, for robustness, in case a sentinel ever ends up inline
// instead.
type anchorNodeRenderer struct {
	bookmarks []bookmarkEntry
}

func (r *anchorNodeRenderer) RegisterFuncs(reg pdflib.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
}

func (r *anchorNodeRenderer) handleSentinel(w *pdflib.Writer, content string) {
	if m := anchorSentinelRe.FindStringSubmatch(content); m != nil {
		w.Pdf.AddInternalLink(m[1])
	}
	if m := bookmarkSentinelRe.FindStringSubmatch(content); m != nil {
		idx, err := strconv.Atoi(m[1])
		if err == nil && idx >= 0 && idx < len(r.bookmarks) {
			entry := r.bookmarks[idx]
			if bm, ok := w.Pdf.(interface {
				Bookmark(text string, level int, y float64)
			}); ok {
				bm.Bookmark(entry.Text, entry.Level, -1)
			}
		}
	}
}

func (r *anchorNodeRenderer) renderHTMLBlock(w *pdflib.Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.HTMLBlock)
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		r.handleSentinel(w, string(seg.Value(source)))
	}
	return ast.WalkContinue, nil
}

func (r *anchorNodeRenderer) renderRawHTML(w *pdflib.Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.RawHTML)
	var content string
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		content += string(seg.Value(source))
	}
	r.handleSentinel(w, content)
	return ast.WalkContinue, nil
}

// textNodeRenderer replaces goldmark-pdf's default ast.KindText renderer
// to fix the soft/hard line-break space loss described in the package
// doc comment: the default renderer writes only Text.Segment's raw bytes
// and never checks Text.SoftLineBreak()/HardLineBreak(), both of which
// represent a space that CommonMark says should render as one but whose
// literal newline byte is deliberately excluded from Segment.
type textNodeRenderer struct{}

func (r *textNodeRenderer) RegisterFuncs(reg pdflib.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindText, r.renderText)
}

func (r *textNodeRenderer) renderText(w *pdflib.Writer, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	text := string(n.Segment.Value(source))
	if n.SoftLineBreak() || n.HardLineBreak() {
		text += " "
	}
	w.WriteText(text)
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
	md, bookmarks := buildMarkdown(book, resolve)
	return newMarkdownPDF(bookmarks).Convert([]byte(md), w)
}

// RenderAll writes one combined PDF covering every book in books, each
// starting on a fresh page under title - the "export everything under
// this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	resolve := makeCombinedResolveFunc(books)
	var b strings.Builder
	var bookmarks []bookmarkEntry
	writeBookmarkSentinel(&b, &bookmarks, title, 0)
	fmt.Fprintf(&b, "# %s\n\n", title)
	for i, book := range books {
		if i > 0 {
			b.WriteString("\n\\newpage\n\n")
		}
		buildMarkdownInto(&b, book, resolve, 1, &bookmarks)
	}
	return newMarkdownPDF(bookmarks).Convert([]byte(b.String()), w)
}

// blockStructureRe matches the first line of a Markdown block that has
// line-significant structure (heading, list item, blockquote, fenced code)
// - normalizeSoftWraps leaves such blocks untouched.
var blockStructureRe = regexp.MustCompile("^(#{1,6}\\s|[-*+]\\s|[0-9]+[.)]\\s|>|```|~~~)")

// normalizeSoftWraps collapses each soft-wrapped line break inside a plain
// prose paragraph into the single space CommonMark says it should render
// as, and leaves everything else (blank-line paragraph boundaries, and
// blocks that look like a heading/list/blockquote/fenced code) untouched.
//
// This exists because textNodeRenderer (above) can only restore a soft
// break's implied space when the break lands at the end of an ast.Text
// node - but when the break instead lands right after a closing link
// (`[text](url)\nmore text`), there is no Text node adjacent to the break
// at all, so goldmark-pdf's renderer has nothing to hang a "this was a
// line break" flag on; the space is unrecoverably gone by render time.
// Confirmed by experiment: "...[testing obligations](url).\n" wrapping
// straight into the *next* sentence rendered as "...obligationsAgents..."
// even with textNodeRenderer active. Doing the normalization here, on the
// raw Markdown source before goldmark ever parses it, sidesteps the gap
// entirely - there is no longer an embedded newline for any inline node
// adjacency to lose track of.
//
// Section commentary is the only input this is applied to: law text
// never reaches here with embedded raw newlines in the first place,
// since internal/parser's fence-aware clause folding already joins a
// law's continuation lines with a single space before it ever becomes
// model.Law.Text.
func normalizeSoftWraps(md string) string {
	blocks := regexp.MustCompile(`\n{2,}`).Split(md, -1)
	for i, block := range blocks {
		if blockStructureRe.MatchString(block) || strings.Contains(block, "```") || strings.Contains(block, "~~~") {
			continue
		}
		lines := strings.Split(block, "\n")
		for j, line := range lines {
			lines[j] = strings.TrimRight(line, " \t")
		}
		blocks[i] = strings.Join(lines, " ")
	}
	return strings.Join(blocks, "\n\n")
}

// buildMarkdown converts a Lawbook IR into a Markdown string that
// goldmark-pdf can render with full CommonMark + GFM support, plus the
// ordered list of /Outlines sidebar entries (see bookmarkNodeRenderer)
// to register alongside it.
func buildMarkdown(book model.Lawbook, resolve ResolveFunc) (string, []bookmarkEntry) {
	var b strings.Builder
	var bookmarks []bookmarkEntry
	buildMarkdownInto(&b, book, resolve, 0, &bookmarks)
	return b.String(), bookmarks
}

// writeBookmarkSentinel registers text as a bookmarks entry at level and
// writes the <!--alaws-bookmark:N--> sentinel referencing it into b, so
// bookmarkNodeRenderer can call Pdf.Bookmark at the right position once
// this Markdown is actually rendered (see anchorSentinelRe's doc comment
// for why the position can only be known at render time, not here).
func writeBookmarkSentinel(b *strings.Builder, bookmarks *[]bookmarkEntry, text string, level int) {
	idx := len(*bookmarks)
	*bookmarks = append(*bookmarks, bookmarkEntry{Text: text, Level: level})
	fmt.Fprintf(b, "<!--alaws-bookmark:%d-->\n", idx)
}

// buildMarkdownInto appends book's Markdown to b, and appends the
// /Outlines sidebar entries for its structural nodes - book title, the
// LawBook/PromptBook group headers (when book has prompts), each section
// heading, and each prompt heading - to *bookmarks. baseLevel is the
// book title's own bookmark level (0 for a standalone Render, 1 under
// RenderAll's combined-title entry); every other level here is derived
// from baseLevel plus, for sections, Section.Level, so the levels handed
// to Pdf.Bookmark are always contiguous (parent, then parent+1) - a gap
// would misattribute PDF outline parents (see gofpdf's putbookmarks,
// which reads outline[level-1] to find each entry's parent).
//
// Deliberately, no bookmark sentinel is ever emitted for a law heading
// or from inside Commentary/Template Markdown: those bodies are written
// through rewriteAlawsLinks/normalizeSoftWraps unchanged below, so a
// "##" a prompt author writes inline for prose formatting (see
// examples/engineering/prompts/code-review.md) renders as an ordinary
// heading with no outline entry of its own.
func buildMarkdownInto(b *strings.Builder, book model.Lawbook, resolve ResolveFunc, baseLevel int, bookmarks *[]bookmarkEntry) {
	writeBookmarkSentinel(b, bookmarks, book.Metadata.Title, baseLevel)
	fmt.Fprintf(b, "# %s\n\n", book.Metadata.Title)

	hasPrompts := len(book.Prompts) > 0
	sectionBase := baseLevel
	if hasPrompts {
		groupLevel := baseLevel + 1
		writeBookmarkSentinel(b, bookmarks, "LawBook", groupLevel)
		b.WriteString("## LawBook\n\n")
		b.WriteString("*Laws and sections that define the book's rules.*\n\n")
		sectionBase = groupLevel
	}

	for _, s := range book.Sections {
		// Section heading
		level := s.Level + 1
		if level > 6 {
			level = 6
		}
		sectionAnchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindSection, Section: s})
		fmt.Fprintf(b, "<!--alaws-anchor:%s-->\n", sectionAnchor)
		writeBookmarkSentinel(b, bookmarks, s.Number+" "+s.Title, sectionBase+s.Level)
		fmt.Fprintf(b, "%s %s %s\n\n", strings.Repeat("#", level), s.Number, s.Title)

		// Section ID as a muted line
		fmt.Fprintf(b, "*%s*\n\n", s.ID)

		// Commentary (already markdown — rewrite alaws: links)
		if s.Commentary != "" {
			b.WriteString(rewriteAlawsLinks(normalizeSoftWraps(s.Commentary), resolve))
			b.WriteString("\n\n")
		}

		// Section-level backlinks
		if bl := book.PromptBacklinks[s.ID]; len(bl) > 0 {
			b.WriteString("**Used in prompts:** ")
			for i, pid := range bl {
				if i > 0 {
					b.WriteString(" · ")
				}
				if href, ok := resolve(pid); ok {
					fmt.Fprintf(b, "[%s](%s)", pid, href)
				} else {
					fmt.Fprintf(b, "%s", pid)
				}
			}
			b.WriteString("\n\n")
		}

		// Laws
		for _, law := range s.Laws {
			anchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
			fmt.Fprintf(b, "<!--alaws-anchor:%s-->\n", anchor)
			fmt.Fprintf(b, "**%s** %s\n\n", law.Number, rewriteAlawsLinks(law.Text, resolve))

			// Per-law backlinks
			lawAnchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
			if bl := book.PromptBacklinks[lawAnchor]; len(bl) > 0 {
				b.WriteString("*Used in:* ")
				for i, pid := range bl {
					if i > 0 {
						b.WriteString(" · ")
					}
					if href, ok := resolve(pid); ok {
						fmt.Fprintf(b, "[%s](%s)", pid, href)
					} else {
						fmt.Fprintf(b, "%s", pid)
					}
				}
				b.WriteString("\n\n")
			}
		}
	}

	// PromptBook section
	if hasPrompts {
		promptGroupLevel := baseLevel + 1
		promptLevel := promptGroupLevel + 1
		b.WriteString("---\n\n")
		writeBookmarkSentinel(b, bookmarks, "PromptBook", promptGroupLevel)
		b.WriteString("## PromptBook\n\n")
		b.WriteString("*Prompt templates that stitch laws and sections into reusable agent prompts.*\n\n")
		for _, p := range book.Prompts {
			promptAnchor := p.ID
			fmt.Fprintf(b, "<!--alaws-anchor:%s-->\n", promptAnchor)
			writeBookmarkSentinel(b, bookmarks, p.Title, promptLevel)
			fmt.Fprintf(b, "### %s\n\n", p.Title)
			fmt.Fprintf(b, "*%s*\n\n", p.ID)

			if p.Commentary != "" {
				b.WriteString(rewriteAlawsLinks(normalizeSoftWraps(p.Commentary), resolve))
				b.WriteString("\n\n")
			}

			// Template content (expanded)
			tmplMD := resolver.PromptDisplayText(p, true)
			b.WriteString(rewriteAlawsLinks(tmplMD, resolve))
			b.WriteString("\n\n")

			// References: sections/laws this prompt pulls in
			if len(p.ReferencedAnchors) > 0 {
				b.WriteString("**References:** ")
				for i, anchor := range p.ReferencedAnchors {
					if i > 0 {
						b.WriteString(" · ")
					}
					if href, ok := resolve(anchor); ok {
						fmt.Fprintf(b, "[%s](%s)", anchor, href)
					} else {
						b.WriteString(anchor)
					}
				}
				b.WriteString("\n\n")
			}
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
