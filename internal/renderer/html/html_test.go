package html

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

// TestRender_NavReflectsIROnly is a regression test for navigation built
// from the compiled Lawbook IR (sections + prompts) rather than from
// scanning rendered Markdown/HTML for heading tags: a prompt's expanded
// template can itself contain Markdown headings written for prose
// formatting (see examples/engineering/prompts/code-review.md's "##
// Mandatory checks" etc.) - those must render as ordinary headings in the
// body, but must never appear as extra <nav class="toc"> entries.
func TestRender_NavReflectsIROnly(t *testing.T) {
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
	out := buf.String()

	navMatch := regexp.MustCompile(`(?s)<nav class="toc"[^>]*>(.*?)</nav>`).FindStringSubmatch(out)
	if navMatch == nil {
		t.Fatalf("no <nav class=\"toc\"> found in output:\n%s", out)
	}
	nav := navMatch[1]

	for _, want := range []string{"Principles", "Review", "Review Prompt"} {
		if !strings.Contains(nav, want) {
			t.Errorf("nav missing expected entry %q; nav:\n%s", want, nav)
		}
	}
	for _, unwanted := range []string{"Mandatory checks", "Output format"} {
		if strings.Contains(nav, unwanted) {
			t.Errorf("nav must not contain prompt-body heading %q (leaked from template content); nav:\n%s", unwanted, nav)
		}
	}

	// The prompt-body headings must still render as real headings in the
	// document body itself - only the nav must exclude them.
	if !strings.Contains(out, "Mandatory checks") || !strings.Contains(out, "Output format") {
		t.Errorf("expected prompt-body headings to still appear in the rendered content")
	}

	navLinks := regexp.MustCompile(`<li><a href=`).FindAllString(nav, -1)
	if len(navLinks) != 3 {
		t.Errorf("nav link count: got %d, want 3 (2 sections + 1 prompt)", len(navLinks))
	}
}
