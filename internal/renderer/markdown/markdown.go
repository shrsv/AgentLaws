// Package markdown renders a compiled Lawbook back to Markdown, from the
// same Lawbook IR as the HTML and PDF renderers (docs/PLAN1.md §22-§23) -
// not a copy of the source files, since it reflects canonical numbering
// and the compiled ordering, not whatever the author originally typed.
package markdown

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/resolver"
)

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

// Render writes the Markdown representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	resolve := makeResolveFunc(book)
	fmt.Fprintf(w, "# %s\n\n", book.Metadata.Title)
	if len(book.Prompts) > 0 {
		fmt.Fprintf(w, "## LawBook\n\n")
		fmt.Fprintf(w, "*Laws and sections that define the book's rules.*\n\n")
	}
	renderSections(w, book, 0, resolve)
	if len(book.Prompts) > 0 {
		renderPrompts(w, book, 0, resolve)
	}
	renderProvenanceFooter(w, book.Provenance)
	return nil
}

// RenderAll writes one combined Markdown document covering every book in
// books, each as its own top-level part under title - the "export
// everything under this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	resolve := makeCombinedResolveFunc(books)
	fmt.Fprintf(w, "# %s\n\n", title)
	for _, book := range books {
		fmt.Fprintf(w, "## %s\n\n", book.Metadata.Title)
		if len(book.Prompts) > 0 {
			fmt.Fprintf(w, "### LawBook\n\n")
			fmt.Fprintf(w, "*Laws and sections that define the book's rules.*\n\n")
		}
		renderSections(w, book, 1, resolve)
		if len(book.Prompts) > 0 {
			renderPrompts(w, book, 1, resolve)
		}
		renderProvenanceFooter(w, book.Provenance)
	}
	return nil
}

func renderSections(w io.Writer, book model.Lawbook, levelOffset int, resolve ResolveFunc) {
	for _, s := range book.Sections {
		level := min(s.Level+1+levelOffset, 6)
		fmt.Fprintf(w, "%s %s %s\n\n", strings.Repeat("#", level), s.Number, s.Title)
		fmt.Fprintf(w, "`%s`\n\n", s.ID)

		if s.Commentary != "" {
			fmt.Fprintf(w, "%s\n\n", rewriteAlawsLinks(s.Commentary, resolve))
		}

		// Section-level backlinks
		if bl := book.PromptBacklinks[s.ID]; len(bl) > 0 {
			fmt.Fprint(w, "**Used in prompts:** ")
			for i, pid := range bl {
				if i > 0 {
					fmt.Fprint(w, " · ")
				}
				fmt.Fprintf(w, "[%s](alaws:%s)", pid, pid)
			}
			fmt.Fprint(w, "\n\n")
		}

		for _, law := range s.Laws {
			anchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
			if anchor != "" {
				fmt.Fprintf(w, "<a id=%q></a>\n", anchor)
			}
			fmt.Fprintf(w, "**%s** %s\n\n", law.Number, rewriteAlawsLinks(law.Text, resolve))

			// Per-law backlinks
			lawAnchor := resolver.AnchorFor(resolver.Resolved{Kind: resolver.KindLaw, Law: law})
			if bl := book.PromptBacklinks[lawAnchor]; len(bl) > 0 {
				fmt.Fprint(w, "*Used in:* ")
				for i, pid := range bl {
					if i > 0 {
						fmt.Fprint(w, " · ")
					}
					fmt.Fprintf(w, "[%s](alaws:%s)", pid, pid)
				}
				fmt.Fprint(w, "\n\n")
			}
		}
	}
}

// renderPrompts writes one book's prompt templates as a "PromptBook" section.
func renderPrompts(w io.Writer, book model.Lawbook, levelOffset int, resolve ResolveFunc) {
	hLevel := min(2+levelOffset, 6)
	fmt.Fprint(w, "---\n\n")
	fmt.Fprintf(w, "%s PromptBook\n\n", strings.Repeat("#", hLevel))
	fmt.Fprint(w, "*Prompt templates that stitch laws and sections into reusable agent prompts.*\n\n")

	for _, p := range book.Prompts {
		pLevel := min(3+levelOffset, 6)
		fmt.Fprintf(w, "%s %s\n\n", strings.Repeat("#", pLevel), p.Title)
		fmt.Fprintf(w, "`%s`\n\n", p.ID)

		if p.Commentary != "" {
			fmt.Fprintf(w, "%s\n\n", rewriteAlawsLinks(p.Commentary, resolve))
		}

		// Template content (expanded)
		tmplMD := resolver.PromptDisplayText(p, true)
		fmt.Fprintf(w, "%s\n\n", rewriteAlawsLinks(tmplMD, resolve))

		// References: sections/laws this prompt pulls in
		if len(p.ReferencedAnchors) > 0 {
			fmt.Fprint(w, "**References:** ")
			for i, anchor := range p.ReferencedAnchors {
				if i > 0 {
					fmt.Fprint(w, " · ")
				}
				if href, ok := resolve(anchor); ok {
					fmt.Fprintf(w, "[%s](%s)", anchor, href)
				} else {
					fmt.Fprint(w, anchor)
				}
			}
			fmt.Fprint(w, "\n\n")
		}
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

// renderProvenanceFooter writes a horizontal rule and provenance metadata
// at the bottom of a compiled Markdown document (docs/PLAN1.md §24-§25,
// §50). Silently omitted when Provenance is empty.
func renderProvenanceFooter(w io.Writer, prov model.Provenance) {
	if prov.Revision == "" && prov.CompiledAt == "" && prov.AgentLawsVersion == "" {
		return
	}
	fmt.Fprint(w, "---\n\n")

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
		parts = append(parts, fmt.Sprintf("revision `%s%s`", short, dirty))
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
	fmt.Fprintf(w, "*%s*\n\n", strings.Join(parts, " · "))
}
