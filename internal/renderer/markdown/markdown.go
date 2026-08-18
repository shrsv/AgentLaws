// Package markdown renders a compiled Lawbook back to Markdown, from the
// same Lawbook IR as the HTML and PDF renderers (docs/PLAN1.md §22-§23) -
// not a copy of the source files, since it reflects canonical numbering
// and the compiled ordering, not whatever the author originally typed.
package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
)

// Render writes the Markdown representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	fmt.Fprintf(w, "# %s\n\n", book.Metadata.Title)
	renderSections(w, book.Sections, 0)
	renderProvenanceFooter(w, book.Provenance)
	return nil
}

// RenderAll writes one combined Markdown document covering every book in
// books, each as its own top-level part under title - the "export
// everything under this root" counterpart to Render (docs/PLAN1.md §57).
func RenderAll(w io.Writer, title string, books []model.Lawbook) error {
	fmt.Fprintf(w, "# %s\n\n", title)
	for _, book := range books {
		fmt.Fprintf(w, "## %s\n\n", book.Metadata.Title)
		renderSections(w, book.Sections, 1)
		renderProvenanceFooter(w, book.Provenance)
	}
	return nil
}

func renderSections(w io.Writer, sections []model.Section, levelOffset int) {
	for _, s := range sections {
		level := min(s.Level+1+levelOffset, 6)
		fmt.Fprintf(w, "%s %s %s\n\n", strings.Repeat("#", level), s.Number, s.Title)
		fmt.Fprintf(w, "`%s`\n\n", s.ID)

		if s.Commentary != "" {
			fmt.Fprintf(w, "%s\n\n", s.Commentary)
		}

		for _, law := range s.Laws {
			// Bold citation prefix, not a Markdown ordered-list marker -
			// a "1. 2.5.3 text" line would let most Markdown renderers'
			// own auto-numbering double up with the citation number, the
			// same bug this fixed in the HTML renderer's <ol>.
			fmt.Fprintf(w, "**%s** %s\n\n", law.Number, law.Text)
		}
	}
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
