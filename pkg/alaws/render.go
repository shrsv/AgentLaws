package alaws

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
	renderhtml "github.com/shrsv/AgentLaws/internal/renderer/html"
	rendermarkdown "github.com/shrsv/AgentLaws/internal/renderer/markdown"
	renderpdf "github.com/shrsv/AgentLaws/internal/renderer/pdf"
)

// RenderHTML writes the human-readable HTML representation of the compiled
// book to w (docs/PLAN1.md §22).
func (b *Book) RenderHTML(w io.Writer) error {
	return renderhtml.Render(w, b.lawbook)
}

// RenderPDF writes the human-readable PDF representation of the compiled
// book to w, from the same Lawbook IR as RenderHTML (docs/PLAN1.md §23).
func (b *Book) RenderPDF(w io.Writer) error {
	return renderpdf.Render(w, b.lawbook)
}

// RenderMarkdown writes the compiled book back out as Markdown, from the
// same Lawbook IR as RenderHTML/RenderPDF - canonical numbering and
// compiled ordering, not a copy of the source files.
func (b *Book) RenderMarkdown(w io.Writer) error {
	return rendermarkdown.Render(w, b.lawbook)
}

// RenderedSection is a section's commentary and each of its laws'
// text, pre-rendered to HTML fragments via the same pipeline RenderHTML
// uses (including syntax-highlighted fenced code blocks). This is what
// lets a UI display properly formatted commentary/laws - inline code,
// lists, code blocks - without shipping its own Markdown parser
// (docs/PLAN1.md §28).
type RenderedSection struct {
	CommentaryHTML string
	LawHTML        map[string]string // law citation number -> HTML
}

// RenderedSections returns every section's commentary and law text
// rendered to HTML fragments, keyed by section ID.
func (b *Book) RenderedSections() (map[string]RenderedSection, error) {
	out := make(map[string]RenderedSection, len(b.lawbook.Sections))
	for _, s := range b.lawbook.Sections {
		commentaryHTML, err := renderhtml.RenderFragment(s.Commentary)
		if err != nil {
			return nil, err
		}
		lawHTML := make(map[string]string, len(s.Laws))
		for _, law := range s.Laws {
			frag, err := renderhtml.RenderFragment(law.Text)
			if err != nil {
				return nil, err
			}
			lawHTML[law.Number] = frag
		}
		out[s.ID] = RenderedSection{CommentaryHTML: commentaryHTML, LawHTML: lawHTML}
	}
	return out, nil
}

// WriteArtifacts renders the book into dir, one file per comma-separated
// format in formats ("html", "json", "pdf", "md" - docs/PLAN1.md §22, §23,
// §26). Every format is a renderer over the same compiled Lawbook IR, not
// a separate parse of the source. This is what `alaws compile` calls; a Go
// caller wanting the same default artifact layout can call it directly.
func (b *Book) WriteArtifacts(dir string, formats string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for f := range strings.SplitSeq(formats, ",") {
		switch strings.TrimSpace(f) {
		case "html":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.html"), b.RenderHTML); err != nil {
				return err
			}
		case "pdf":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.pdf"), b.RenderPDF); err != nil {
				return err
			}
		case "md":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.md"), b.RenderMarkdown); err != nil {
				return err
			}
		case "json":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.json"), func(w io.Writer) error {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(b.lawbook)
			}); err != nil {
				return err
			}
		case "":
			// allow trailing commas
		default:
			return fmt.Errorf("unknown artifact format %q", f)
		}
	}
	return nil
}

func writeArtifactFile(path string, render func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return render(f)
}

// CompileAll discovers every book under root and compiles each one. It
// always returns one *Book per discovered book (in Discover's order,
// docs/PLAN1.md §57), even when some failed to compile - each failed
// book's own Diagnostics/error still describes what went wrong, the same
// way Compile does for one book. The returned error, if any, joins every
// per-book compile error so a caller can report all of them, not just the
// first (errors.Is/As still work against it via errors.Join).
func CompileAll(root string) ([]*Book, error) {
	infos, err := Discover(root)
	if err != nil {
		return nil, err
	}
	books := make([]*Book, 0, len(infos))
	var errs []error
	for _, info := range infos {
		b, err := Compile(info.Path)
		books = append(books, b)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", info.Path, err))
		}
	}
	return books, errors.Join(errs...)
}

func lawbooksOf(books []*Book) []model.Lawbook {
	out := make([]model.Lawbook, len(books))
	for i, b := range books {
		out[i] = b.lawbook
	}
	return out
}

// RenderCombinedHTML writes one HTML document covering every book in
// books, each as its own top-level part - the "export the whole thing"
// counterpart to Book.RenderHTML (docs/PLAN1.md §57).
func RenderCombinedHTML(w io.Writer, title string, books []*Book) error {
	return renderhtml.RenderAll(w, title, lawbooksOf(books))
}

// RenderCombinedPDF writes one PDF covering every book in books, each
// starting on a fresh page - the "export the whole thing" counterpart to
// Book.RenderPDF.
func RenderCombinedPDF(w io.Writer, title string, books []*Book) error {
	return renderpdf.RenderAll(w, title, lawbooksOf(books))
}

// RenderCombinedMarkdown writes one Markdown document covering every book
// in books, each as its own top-level part - the "export the whole thing"
// counterpart to Book.RenderMarkdown.
func RenderCombinedMarkdown(w io.Writer, title string, books []*Book) error {
	return rendermarkdown.RenderAll(w, title, lawbooksOf(books))
}

// WriteCombinedArtifacts renders every book in books into dir as one
// combined document per comma-separated format in formats ("html", "pdf",
// "md" - "json" isn't offered here since a bare array of Lawbooks is
// already what a caller wanting per-book JSON should get from CompileAll
// directly, rather than a combined-file convention this package would
// have to invent).
func WriteCombinedArtifacts(books []*Book, title string, dir string, formats string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for f := range strings.SplitSeq(formats, ",") {
		switch strings.TrimSpace(f) {
		case "html":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.html"), func(w io.Writer) error {
				return RenderCombinedHTML(w, title, books)
			}); err != nil {
				return err
			}
		case "pdf":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.pdf"), func(w io.Writer) error {
				return RenderCombinedPDF(w, title, books)
			}); err != nil {
				return err
			}
		case "md":
			if err := writeArtifactFile(filepath.Join(dir, "lawbook.md"), func(w io.Writer) error {
				return RenderCombinedMarkdown(w, title, books)
			}); err != nil {
				return err
			}
		case "":
			// allow trailing commas
		default:
			return fmt.Errorf("unknown combined artifact format %q (want html, pdf, or md)", f)
		}
	}
	return nil
}
