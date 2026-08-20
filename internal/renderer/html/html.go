// Package html renders a compiled Lawbook to a human-readable HTML document.
// It operates on the Lawbook IR only, never on Markdown directly (PLAN1
// §22-§23). Commentary and law text are themselves Markdown (docs/PLAN1.md
// §7) and are rendered through goldmark, including fenced code blocks with
// syntax highlighting - not dumped as plain text.
package html

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/renderer/navtree"
	"github.com/shrsv/AgentLaws/internal/resolver"
)

// ResolveFunc resolves an alaws: link token to an href. Returns the href
// and true on success, or ("", false) if the token didn't resolve.
type ResolveFunc func(token string) (href string, ok bool)

// markdown is the single goldmark instance every renderer in this package
// uses - RenderFragment, and by extension the web UI (via
// pkg/alaws.RenderedSections), stays visually identical to the static HTML
// export because both go through this same configuration.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("monokai"),
		),
	),
)

// RenderFragment converts a snippet of Markdown - a law's text or a
// section's commentary - to an HTML fragment, using the same
// configuration (including syntax highlighting) as Render/RenderAll. This
// is what lets the web UI show properly formatted commentary/laws without
// shipping its own Markdown parser: it asks for this fragment through the
// API instead (docs/PLAN1.md §28).
//
// If resolve is non-nil, any alaws: link destinations are rewritten using
// it. If resolve is nil, alaws: links are left untouched.
func RenderFragment(md string, resolve ResolveFunc) (string, error) {
	if resolve == nil {
		var buf bytes.Buffer
		if err := markdown.Convert([]byte(md), &buf); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	// Parse, rewrite alaws: links, then render.
	reader := text.NewReader([]byte(md))
	parsed := markdown.Parser().Parse(reader)

	// Walk AST and rewrite alaws: link destinations.
	ast.Walk(parsed, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		dest := string(link.Destination)
		if !strings.HasPrefix(dest, "alaws:") {
			return ast.WalkContinue, nil
		}
		token := strings.TrimPrefix(dest, "alaws:")
		if href, ok := resolve(token); ok {
			link.Destination = []byte(href)
		}
		return ast.WalkContinue, nil
	})

	var buf bytes.Buffer
	if err := markdown.Renderer().Render(&buf, []byte(md), parsed); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const style = `<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;margin:0;padding:0;color:#1e1e1e;line-height:1.55}
.layout{display:grid;grid-template-columns:260px 1fr;align-items:start;max-width:1180px;margin:0 auto}
nav.toc{position:sticky;top:0;max-height:100vh;overflow-y:auto;padding:2rem 1rem 2rem 1.5rem;border-right:1px solid #ddd;box-sizing:border-box;font-size:.85rem}
nav.toc .toc-book+.toc-book{margin-top:1.5rem}
nav.toc .toc-book-title{font-weight:700;margin:0 0 .4rem}
nav.toc .toc-group-title{font-weight:600;color:#555;margin:.9rem 0 .3rem;font-size:.78rem;text-transform:uppercase;letter-spacing:.04em}
nav.toc ul{list-style:none;margin:0;padding-left:0}
nav.toc ul.toc-children{padding-left:.9rem}
nav.toc li{margin:.15rem 0}
nav.toc a{color:#1e1e1e;text-decoration:none;display:block;padding:.1rem 0}
nav.toc a:hover{color:#1a73e8;text-decoration:underline}
nav.toc .toc-number{color:#767676;font-family:ui-monospace,Menlo,monospace;font-size:.8em;margin-right:.35em}
.content{min-width:0;max-width:860px;margin:2rem auto;padding:0 1rem;box-sizing:border-box}
@media (max-width:700px){.layout{display:block}nav.toc{position:static;max-height:none;border-right:none;border-bottom:1px solid #ddd}}
h1{border-bottom:1px solid #ddd;padding-bottom:.5rem}
h2.book-title{margin-top:3rem;border-bottom:2px solid #333;padding-bottom:.4rem}
.section-id{color:#767676;font-family:ui-monospace,Menlo,monospace;font-size:.85rem;margin-top:-.5rem}
ol.laws{padding-left:1.4rem;list-style:none}
ol.laws>li{margin:.4rem 0}
ol.laws>li p{display:inline;margin:0}
.law-number{color:#098658;font-family:ui-monospace,Menlo,monospace;margin-right:.4rem}
code{font-family:ui-monospace,Menlo,monospace;background:#f0f0f0;padding:.1em .3em;border-radius:3px;font-size:.92em}
pre{overflow-x:auto;border:1px solid #ddd;border-radius:6px;padding:.85rem 1rem;font-size:.85rem;line-height:1.5;background:#272822}
pre code{background:none;padding:0;border-radius:0;font-size:1em;color:#f8f8f2}
footer.provenance{margin-top:1.5rem;padding-top:.6rem;border-top:1px solid #ddd;color:#767676;font-size:.8rem}
table{border-collapse:collapse;width:100%;margin:1rem 0;font-size:.92em}
th,td{border:1px solid #ddd;padding:.5rem .75rem;text-align:left}
th{background:#f5f5f5;font-weight:600}
tr:nth-child(even){background:#fafafa}
hr.promptbook-separator{margin:3rem 0 1rem;border:none;border-top:2px solid #333}
.promptbook-subtitle{color:#767676;font-size:.9rem;margin-top:-.4rem;margin-bottom:1.5rem}
.prompt-template-content{background:#f8f8f8;border:1px solid #ddd;border-radius:6px;padding:.85rem 1rem;margin:.5rem 0;font-size:.92rem;line-height:1.55}
.prompt-template-content code{font-family:ui-monospace,Menlo,monospace;background:#f0f0f0;padding:.1em .3em;border-radius:3px;font-size:.92em}
.prompt-template-content pre{background:#272822}
.prompt-template-content pre code{background:none;padding:0;border-radius:0;color:#f8f8f2}
.backlinks{color:#767676;font-size:.85rem;margin:.25rem 0}
.backlinks-label{font-weight:600}
.law-backlinks{color:#767676;font-size:.8rem}
.alaws-link{color:#1a73e8;text-decoration:none}
.alaws-link:hover{text-decoration:underline}
</style>`

// Render writes the HTML representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	resolve := makeResolveFunc(book, "")
	fmt.Fprintf(w, "<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>%s</title>%s</head><body>\n",
		html.EscapeString(book.Metadata.Title), style)
	fmt.Fprint(w, "<div class=\"layout\">\n<nav class=\"toc\" aria-label=\"Table of contents\">\n")
	renderNavBook(w, book, "")
	fmt.Fprint(w, "</nav>\n<div class=\"content\">\n")
	fmt.Fprintf(w, "<h1>%s</h1>\n", html.EscapeString(book.Metadata.Title))
	if len(book.Prompts) > 0 {
		fmt.Fprint(w, "<h2 id=\"laws\">LawBook</h2>\n")
		fmt.Fprint(w, "<p class=\"promptbook-subtitle\">Laws and sections that define the book's rules.</p>\n")
	}
	if err := renderSections(w, book, "", 0, resolve); err != nil {
		return err
	}
	if len(book.Prompts) > 0 {
		if err := renderPrompts(w, book, "", 0, resolve); err != nil {
			return err
		}
	}
	renderProvenanceFooter(w, book.Provenance)
	fmt.Fprint(w, "</div>\n</div>\n</body></html>\n")
	return nil
}

// RenderAll writes one combined HTML document covering every book in
// books, each as its own top-level part under title - the "export
// everything under this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	// Build a combined resolver that searches all books.
	combinedResolve := func(token string) (string, bool) {
		for i, book := range books {
			idPrefix := fmt.Sprintf("book%d-", i)
			resolve := makeResolveFunc(book, idPrefix)
			if href, ok := resolve(token); ok {
				return href, true
			}
		}
		return "", false
	}

	fmt.Fprintf(w, "<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>%s</title>%s</head><body>\n",
		html.EscapeString(title), style)
	fmt.Fprint(w, "<div class=\"layout\">\n<nav class=\"toc\" aria-label=\"Table of contents\">\n")
	for i, book := range books {
		idPrefix := fmt.Sprintf("book%d-", i)
		fmt.Fprint(w, "<div class=\"toc-book\">\n")
		fmt.Fprintf(w, "<p class=\"toc-book-title\">%s</p>\n", html.EscapeString(book.Metadata.Title))
		renderNavBook(w, book, idPrefix)
		fmt.Fprint(w, "</div>\n")
	}
	fmt.Fprint(w, "</nav>\n<div class=\"content\">\n")
	fmt.Fprintf(w, "<h1>%s</h1>\n", html.EscapeString(title))
	for i, book := range books {
		fmt.Fprintf(w, "<h2 class=\"book-title\">%s</h2>\n", html.EscapeString(book.Metadata.Title))
		idPrefix := fmt.Sprintf("book%d-", i)
		if len(book.Prompts) > 0 {
			fmt.Fprintf(w, "<h3 id=%q>LawBook</h3>\n", html.EscapeString(idPrefix+"laws"))
			fmt.Fprintf(w, "<p class=\"promptbook-subtitle\">Laws and sections that define the book's rules.</p>\n")
		}
		if err := renderSections(w, book, idPrefix, 1, combinedResolve); err != nil {
			return err
		}
		if len(book.Prompts) > 0 {
			if err := renderPrompts(w, book, idPrefix, 1, combinedResolve); err != nil {
				return err
			}
		}
		renderProvenanceFooter(w, book.Provenance)
	}
	fmt.Fprint(w, "</div>\n</div>\n</body></html>\n")
	return nil
}

// renderNavBook writes one book's entry into the <nav class="toc"> block:
// a nested list of its sections, matching the hierarchy alaws UI's own
// sidebar shows (buildTree in web/src/views/BookDetail.tsx), followed by
// a flat list of its prompts, if any. Built entirely from the compiled
// IR - never by scanning rendered Markdown/HTML for heading tags - so
// headings an author writes inside commentary or prompt bodies for prose
// formatting can never leak into it.
func renderNavBook(w io.Writer, book model.Lawbook, idPrefix string) {
	roots := navtree.Build(book.Sections)
	if len(roots) > 0 {
		// Only labeled when there's a PromptBook group to distinguish it
		// from, matching the in-body "LawBook" heading (renderSections'
		// caller), which is likewise only shown in that same case.
		if len(book.Prompts) > 0 {
			fmt.Fprint(w, "<p class=\"toc-group-title\">LawBook</p>\n")
		}
		fmt.Fprint(w, "<ul class=\"toc-sections\">\n")
		renderNavNodes(w, roots, idPrefix)
		fmt.Fprint(w, "</ul>\n")
	}
	if len(book.Prompts) > 0 {
		fmt.Fprint(w, "<p class=\"toc-group-title\">PromptBook</p>\n<ul class=\"toc-prompts\">\n")
		for _, p := range book.Prompts {
			fmt.Fprintf(w, "<li><a href=%q>%s</a></li>\n",
				html.EscapeString("#"+idPrefix+p.ID), html.EscapeString(p.Title))
		}
		fmt.Fprint(w, "</ul>\n")
	}
}

func renderNavNodes(w io.Writer, nodes []*navtree.Node, idPrefix string) {
	for _, n := range nodes {
		s := n.Section
		fmt.Fprintf(w, "<li><a href=%q><span class=\"toc-number\">%s</span>%s</a>",
			html.EscapeString("#"+idPrefix+s.ID), html.EscapeString(s.Number), html.EscapeString(s.Title))
		if len(n.Children) > 0 {
			fmt.Fprint(w, "<ul class=\"toc-children\">\n")
			renderNavNodes(w, n.Children, idPrefix)
			fmt.Fprint(w, "</ul>\n")
		}
		fmt.Fprint(w, "</li>\n")
	}
}

// makeResolveFunc builds a ResolveFunc for a single book.
func makeResolveFunc(book model.Lawbook, idPrefix string) ResolveFunc {
	return func(token string) (string, bool) {
		r, err := resolver.Resolve(book, token)
		if err != nil {
			return "", false
		}
		return "#" + idPrefix + resolver.AnchorFor(r), true
	}
}

// renderSections writes one book's sections. idPrefix disambiguates anchor
// IDs when several books share one document (RenderAll); levelOffset
// shifts heading levels down to make room for a book-title heading above
// them in that same case.
func renderSections(w io.Writer, book model.Lawbook, idPrefix string, levelOffset int, resolve ResolveFunc) error {
	for _, s := range book.Sections {
		level := min(s.Level+1+levelOffset, 6)
		anchorID := idPrefix + s.ID
		fmt.Fprintf(w, "<h%d id=%q>%s %s</h%d>\n", level, html.EscapeString(anchorID),
			html.EscapeString(s.Number), html.EscapeString(s.Title), level)
		fmt.Fprintf(w, "<p class=\"section-id\">%s</p>\n", html.EscapeString(s.ID))

		if s.Commentary != "" {
			frag, err := RenderFragment(s.Commentary, resolve)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(w, frag); err != nil {
				return err
			}
		}

		// Section-level backlinks
		if bl := book.PromptBacklinks[s.ID]; len(bl) > 0 {
			fmt.Fprint(w, "<p class=\"backlinks\"><span class=\"backlinks-label\">Used in prompts:</span> ")
			for i, pid := range bl {
				if i > 0 {
					fmt.Fprint(w, " · ")
				}
				fmt.Fprintf(w, "<a href=\"%s\" class=\"alaws-link\">%s</a>",
					html.EscapeString("#"+idPrefix+pid), html.EscapeString(pid))
			}
			fmt.Fprint(w, "</p>\n")
		}

		if len(s.Laws) > 0 {
			fmt.Fprint(w, "<ol class=\"laws\">\n")
			for _, law := range s.Laws {
				frag, err := RenderFragment(law.Text, resolve)
				if err != nil {
					return err
				}
				anchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
				fmt.Fprintf(w, "<li id=%q><span class=\"law-number\">%s</span>%s",
					html.EscapeString(idPrefix+anchor), html.EscapeString(law.Number), frag)

				// Per-law backlinks
				lawAnchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
				if bl := book.PromptBacklinks[lawAnchor]; len(bl) > 0 {
					fmt.Fprint(w, "<br><span class=\"law-backlinks\">Used in: ")
					for i, pid := range bl {
						if i > 0 {
							fmt.Fprint(w, " · ")
						}
						fmt.Fprintf(w, "<a href=\"%s\" class=\"alaws-link\">%s</a>",
							html.EscapeString("#"+idPrefix+pid), html.EscapeString(pid))
					}
					fmt.Fprint(w, "</span>")
				}

				fmt.Fprint(w, "</li>\n")
			}
			fmt.Fprint(w, "</ol>\n")
		}
	}
	return nil
}

// renderPrompts writes one book's prompt templates as a "PromptBook" section.
func renderPrompts(w io.Writer, book model.Lawbook, idPrefix string, levelOffset int, resolve ResolveFunc) error {
	hLevel := min(2+levelOffset, 6)
	fmt.Fprintf(w, "<hr class=\"promptbook-separator\">\n")
	fmt.Fprintf(w, "<h%d id=%q>PromptBook</h%d>\n", hLevel, html.EscapeString(idPrefix+"prompts"), hLevel)
	fmt.Fprintf(w, "<p class=\"promptbook-subtitle\">Prompt templates that stitch laws and sections into reusable agent prompts.</p>\n")

	for _, p := range book.Prompts {
		pLevel := min(3+levelOffset, 6)
		anchorID := idPrefix + p.ID
		fmt.Fprintf(w, "<h%d id=%q>%s</h%d>\n", pLevel, html.EscapeString(anchorID),
			html.EscapeString(p.Title), pLevel)
		fmt.Fprintf(w, "<p class=\"section-id\">%s</p>\n", html.EscapeString(p.ID))

		if p.Commentary != "" {
			frag, err := RenderFragment(p.Commentary, resolve)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(w, frag); err != nil {
				return err
			}
		}

		// Template content (expanded)
		tmplMD := resolver.PromptDisplayText(p, true)
		tmplFrag, err := RenderFragment(tmplMD, resolve)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "<div class=\"prompt-template-content\">%s</div>\n", tmplFrag)

		// References: sections/laws this prompt pulls in
		if len(p.ReferencedAnchors) > 0 {
			fmt.Fprint(w, "<p class=\"backlinks\"><span class=\"backlinks-label\">References:</span> ")
			for i, anchor := range p.ReferencedAnchors {
				if i > 0 {
					fmt.Fprint(w, " · ")
				}
				if href, ok := resolve(anchor); ok {
					fmt.Fprintf(w, "<a href=\"%s\" class=\"alaws-link\">%s</a>",
						html.EscapeString(href), html.EscapeString(anchor))
				} else {
					fmt.Fprintf(w, "%s", html.EscapeString(anchor))
				}
			}
			fmt.Fprint(w, "</p>\n")
		}
	}
	return nil
}

// renderProvenanceFooter writes a <footer> with provenance metadata from the
// compiled lawbook (docs/PLAN1.md §24-§25, §50). Silently omitted when
// Provenance is empty (e.g. the book was never compiled through pkg/alaws).
func renderProvenanceFooter(w io.Writer, prov model.Provenance) {
	if prov.Revision == "" && prov.CompiledAt == "" && prov.AgentLawsVersion == "" {
		return
	}
	fmt.Fprint(w, "<footer class=\"provenance\">\n")

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
	fmt.Fprintf(w, "<p>%s</p>\n", html.EscapeString(strings.Join(parts, " · ")))
	fmt.Fprint(w, "</footer>\n")
}
