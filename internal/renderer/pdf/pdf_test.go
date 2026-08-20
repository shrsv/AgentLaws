package pdf

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

// TestRender_InternalLinksAndSpacing is a regression test for two
// goldmark-pdf@v0.4.2 bugs fixed in this package (see the package doc
// comment, fixed_fpdf.go, and normalizeSoftWraps): internal alaws: links
// used to render as broken, double-annotated links, and a commentary
// paragraph that wraps across source lines used to lose the space at
// each wrap point once it crossed a link boundary.
func TestRender_InternalLinksAndSpacing(t *testing.T) {
	book := model.Lawbook{
		Metadata: model.LawbookMetadata{Title: "Test Book"},
		Sections: []model.Section{
			{
				ID:     "principles",
				Number: "1",
				Title:  "Principles",
				Level:  1,
				Commentary: "This is complemented by\n" +
					"[review requirements](alaws:principles.review-required)\n" +
					"and more text after it.",
				Laws: []model.Law{
					{Number: "1.1", Index: 1, Text: "Small changes are reviewable.", Slug: "review-required", SectionID: "principles"},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, book); err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := buf.String()

	linkAnnots := regexp.MustCompile(`/Subtype\s*/Link`).FindAllString(raw, -1)
	uriAnnots := regexp.MustCompile(`/S\s*/URI`).FindAllString(raw, -1)
	// Scoped to /Subtype /Link annotation objects specifically - the
	// document also now carries /Dest entries for its /Outlines sidebar
	// bookmarks (book title + section, one apiece for this book), which
	// are a separate, intentional PDF construct and not what this test
	// is checking.
	linkDestAnnots := linkDestRe.FindAllString(raw, -1)
	if len(linkAnnots) != 1 || len(uriAnnots) != 0 || len(linkDestAnnots) != 1 {
		t.Errorf("internal link annotations: got annots=%d uri=%d linkDest=%d, want exactly one clean /Dest link and zero URI actions",
			len(linkAnnots), len(uriAnnots), len(linkDestAnnots))
	}
}

// linkDestRe matches a /Dest entry that belongs to a /Subtype /Link
// annotation object specifically, as opposed to one belonging to an
// /Outlines sidebar bookmark entry - both are legitimate uses of /Dest
// in the documents this package renders.
var linkDestRe = regexp.MustCompile(`(?s)/Subtype\s*/Link.*?/Dest\s*\[`)

// TestRender_SectionLevelInternalLink is a regression test for a bug found
// while regenerating a real lawbook's PDF: buildMarkdownInto only ever
// emitted an alaws-anchor sentinel for laws, never for section headings, so
// any alaws: link whose target was a section (not a law) resolved to a
// href with no matching registered anchor - fixedFpdf.Write silently drops
// such a pending link (see its "Dangling alaws: reference" comment) rather
// than panicking, so the link vanished from the PDF with no error at all.
func TestRender_SectionLevelInternalLink(t *testing.T) {
	book := model.Lawbook{
		Metadata: model.LawbookMetadata{Title: "Test Book"},
		Sections: []model.Section{
			{
				ID:         "intro",
				Number:     "1",
				Title:      "Intro",
				Level:      1,
				Commentary: "See [the principles section](alaws:principles) for more.",
			},
			{
				ID:     "principles",
				Number: "2",
				Title:  "Principles",
				Level:  1,
				Laws: []model.Law{
					{Number: "2.1", Index: 1, Text: "Small changes are reviewable.", Slug: "review-required", SectionID: "principles"},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, book); err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := buf.String()

	linkDestAnnots := linkDestRe.FindAllString(raw, -1)
	if len(linkDestAnnots) != 1 {
		t.Errorf("got %d link /Dest annotations, want exactly 1 for the section-level alaws: link", len(linkDestAnnots))
	}
}

// TestRender_OutlineReflectsIROnly is a regression test for the
// /Outlines sidebar (bookmarkNodeRenderer/writeBookmarkSentinel):
// entries must come straight from the compiled Lawbook IR - book title,
// LawBook/PromptBook group headers, sections (nested by Level), and
// prompts - never from scanning rendered Markdown for heading tags. A
// prompt's own expanded template can contain "##" headings an author
// wrote for prose formatting (see
// examples/engineering/prompts/code-review.md's "## Mandatory checks"
// etc.); those must render as ordinary headings in the body but must
// never add extra outline entries.
func TestRender_OutlineReflectsIROnly(t *testing.T) {
	book := model.Lawbook{
		Metadata: model.LawbookMetadata{Title: "Test Book"},
		Sections: []model.Section{
			{ID: "principles", Number: "1", Title: "Principles", Level: 1},
			{ID: "principles.review", Number: "1.1", Title: "Review", Level: 2},
		},
		Prompts: []model.PromptTemplate{
			{
				ID:    "review-prompt",
				Title: "Review Prompt",
				Segments: []model.PromptSegment{{
					Kind: model.SegmentText,
					Text: "Do the review.\n\n## Mandatory checks\n\nCheck things.\n\n" +
						"## Output format\n\nSay things.",
				}},
			},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, book); err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := buf.String()

	if !regexp.MustCompile(`/Type\s*/Outlines`).MatchString(raw) {
		t.Fatalf("expected a /Type /Outlines root in the PDF")
	}

	titleObjs := regexp.MustCompile(`<</Title `).FindAllString(raw, -1)
	// book title, LawBook group, 2 sections, PromptBook group, 1 prompt.
	const want = 6
	if len(titleObjs) != want {
		t.Errorf("outline entry count: got %d, want %d (book + LawBook + 2 sections + PromptBook + 1 prompt) - "+
			"prompt-body \"##\" headings must not add extra entries", len(titleObjs), want)
	}
}

func TestNormalizeSoftWraps(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "wrapped prose collapses to single spaces",
			in:   "This is complemented by\n[a link](alaws:x)\nand more text.",
			want: "This is complemented by [a link](alaws:x) and more text.",
		},
		{
			name: "blank-line paragraph boundary preserved",
			in:   "First paragraph\nwraps here.\n\nSecond paragraph\nalso wraps.",
			want: "First paragraph wraps here.\n\nSecond paragraph also wraps.",
		},
		{
			name: "fenced code block left untouched",
			in:   "Some text.\n\n```\nline one\nline two\n```",
			want: "Some text.\n\n```\nline one\nline two\n```",
		},
		{
			name: "bullet list left untouched",
			in:   "- item one\n- item two",
			want: "- item one\n- item two",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSoftWraps(tt.in)
			if got != tt.want {
				t.Errorf("normalizeSoftWraps(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
