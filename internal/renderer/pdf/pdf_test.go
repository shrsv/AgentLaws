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
	destAnnots := regexp.MustCompile(`/Dest\s*\[`).FindAllString(raw, -1)
	if len(linkAnnots) != 1 || len(uriAnnots) != 0 || len(destAnnots) != 1 {
		t.Errorf("internal link annotations: got annots=%d uri=%d dest=%d, want exactly one clean /Dest link and zero URI actions",
			len(linkAnnots), len(uriAnnots), len(destAnnots))
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
