// Package alaws is the public Go API for loading, compiling, resolving, and
// extracting laws from an AgentLaws lawbook. See docs/PLAN1.md §51 for the
// intended API surface: applications should be able to ask questions such
// as book.Resolve, book.Section, book.Laws, and book.History without
// understanding the filesystem parser.
//
// The implementation currently delegates to internal packages that are
// stubs (see docs/PLAN1.md §64 for the milestone sequence); this package
// fixes the public signatures those milestones must satisfy.
package alaws

import (
	"strings"

	"github.com/athreyac4/agentlaws/internal/compiler"
	"github.com/athreyac4/agentlaws/internal/model"
	"github.com/athreyac4/agentlaws/internal/resolver"
	"github.com/athreyac4/agentlaws/internal/template"
)

// Book wraps a compiled Lawbook and exposes the library's query surface.
type Book struct {
	lawbook model.Lawbook
}

// Load compiles and loads the lawbook cluster at path.
func Load(path string) (*Book, error) {
	result, err := compiler.Compile(path, compiler.Options{})
	if err != nil {
		return nil, err
	}
	return &Book{lawbook: result.Lawbook}, nil
}

// Resolve resolves a canonical citation such as "2.5.3" to its Law.
func (b *Book) Resolve(citation string) (model.Law, error) {
	return resolver.ResolveLaw(b.lawbook, citation)
}

// Section resolves a stable section ID such as "engineering.security" to
// its Section.
func (b *Book) Section(id string) (model.Section, error) {
	return resolver.ResolveSection(b.lawbook, id)
}

// Selector picks a subset of a Book's laws, e.g. by section ID or citation.
// The zero value selects nothing; use Sections/Citations/All to build one.
type Selector struct {
	SectionIDs []string
	Citations  []string
	All        bool
}

// LawSet is a selected, orderable set of laws ready for extraction into an
// agent prompt (docs/PLAN1.md §16).
type LawSet struct {
	Laws []model.Law
}

// Laws selects laws from the book per sel: every law in the book (All),
// every law in the given sections (SectionIDs), and/or individually cited
// laws (Citations). Selected laws are returned in book order, section by
// section, deduplicated by citation.
func (b *Book) Laws(sel Selector) (LawSet, error) {
	var out []model.Law
	seen := map[string]bool{}

	add := func(l model.Law) {
		if !seen[l.Number] {
			seen[l.Number] = true
			out = append(out, l)
		}
	}

	if sel.All {
		for _, s := range b.lawbook.Sections {
			for _, l := range s.Laws {
				add(l)
			}
		}
	}

	if len(sel.SectionIDs) > 0 {
		wanted := map[string]bool{}
		for _, id := range sel.SectionIDs {
			wanted[id] = true
		}
		for _, s := range b.lawbook.Sections {
			if wanted[s.ID] {
				for _, l := range s.Laws {
					add(l)
				}
			}
		}
	}

	for _, citation := range sel.Citations {
		law, err := resolver.ResolveLaw(b.lawbook, citation)
		if err != nil {
			return LawSet{}, err
		}
		add(law)
	}

	return LawSet{Laws: out}, nil
}

// MissingPolicy controls Render's behavior for a placeholder with no value.
// It mirrors internal/template.MissingPolicy (docs/PLAN1.md §17a).
type MissingPolicy = template.MissingPolicy

const (
	MissingError           = template.MissingError
	MissingKeepPlaceholder = template.MissingKeepPlaceholder
	MissingEmpty           = template.MissingEmpty
)

// RenderOptions configures LawSet.Render.
type RenderOptions struct {
	Vars      map[string]string
	OnMissing MissingPolicy
}

// Render renders the selected laws as prompt-ready text, substituting
// {{variable}} placeholders per opts (docs/PLAN1.md §17a). It never mutates
// the underlying compiled Lawbook.
func (ls LawSet) Render(opts RenderOptions) (string, error) {
	var out strings.Builder
	for i, law := range ls.Laws {
		rendered, err := template.Render(law.Text, opts.Vars, opts.OnMissing)
		if err != nil {
			return "", err
		}
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(law.Number)
		out.WriteByte(' ')
		out.WriteString(rendered)
	}
	return out.String(), nil
}
