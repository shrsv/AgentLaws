package promptexpand

import (
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/parser"
	"github.com/shrsv/AgentLaws/internal/validator"
)

func bookFixture() model.Lawbook {
	return model.Lawbook{
		Sections: []model.Section{
			{
				ID:     "engineering.coding",
				Number: "1",
				Title:  "Coding",
				Laws: []model.Law{
					{Number: "1.1", Text: "Run tests before proposing a change.", Slug: "run-tests", SectionID: "engineering.coding"},
					{Number: "1.2", Text: "Generated code must be reviewed.", Slug: "review-generated", SectionID: "engineering.coding"},
				},
			},
			{
				ID:     "engineering.security",
				Number: "2",
				Title:  "Security",
				Laws: []model.Law{
					{Number: "2.1", Text: "No secrets in SCM.", Slug: "no-secrets-in-scm", SectionID: "engineering.security"},
				},
			},
		},
	}
}

func TestExpand_LawRef(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "test.prompt",
			Title:       "Test Prompt",
			Commentary:  "A test prompt.",
			RawTemplate: "Apply: {{ref:engineering.coding.run-tests}}",
			Source:      model.SourceRef{Path: "test.md"},
		},
	}

	prompts, backlinks, diags := Expand(book, raw)
	if validator.HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(prompts) != 1 {
		t.Fatalf("got %d prompts, want 1", len(prompts))
	}

	pt := prompts[0]
	if pt.ID != "test.prompt" {
		t.Errorf("ID: got %q", pt.ID)
	}
	if len(pt.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(pt.Segments))
	}
	if pt.Segments[0].Kind != model.SegmentText {
		t.Errorf("seg[0].Kind: got %v, want SegmentText", pt.Segments[0].Kind)
	}
	if pt.Segments[1].Kind != model.SegmentLawRef {
		t.Errorf("seg[1].Kind: got %v, want SegmentLawRef", pt.Segments[1].Kind)
	}
	if pt.Segments[1].Expanded != "1.1 Run tests before proposing a change." {
		t.Errorf("seg[1].Expanded: got %q", pt.Segments[1].Expanded)
	}
	if pt.Segments[1].RefAnchor != "engineering.coding.run-tests" {
		t.Errorf("seg[1].RefAnchor: got %q", pt.Segments[1].RefAnchor)
	}

	// Check backlinks
	if bl, ok := backlinks["engineering.coding.run-tests"]; !ok || len(bl) != 1 || bl[0] != "test.prompt" {
		t.Errorf("backlinks: got %v", backlinks)
	}
}

func TestExpand_SectionRef(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "test.prompt",
			Title:       "Test Prompt",
			Commentary:  "A test prompt.",
			RawTemplate: "Apply:\n{{ref:engineering.coding}}",
			Source:      model.SourceRef{Path: "test.md"},
		},
	}

	prompts, _, diags := Expand(book, raw)
	if validator.HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(prompts) != 1 {
		t.Fatalf("got %d prompts, want 1", len(prompts))
	}

	pt := prompts[0]
	if len(pt.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(pt.Segments))
	}
	if pt.Segments[1].Kind != model.SegmentSectionRef {
		t.Errorf("seg[1].Kind: got %v, want SegmentSectionRef", pt.Segments[1].Kind)
	}
	if pt.Segments[1].RefLabel != "1 Coding" {
		t.Errorf("seg[1].RefLabel: got %q", pt.Segments[1].RefLabel)
	}
	// Section expansion should include title and both laws
	expanded := pt.Segments[1].Expanded
	if expanded != "1 Coding:\n\n1.1 Run tests before proposing a change.\n\n1.2 Generated code must be reviewed." {
		t.Errorf("seg[1].Expanded: got %q", expanded)
	}
}

func TestExpand_SectionRef_NoTitle(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "test.prompt",
			Title:       "Test Prompt",
			Commentary:  "A test prompt.",
			RawTemplate: "Apply:\n{{ref:engineering.coding#notitle}}",
			Source:      model.SourceRef{Path: "test.md"},
		},
	}

	prompts, _, diags := Expand(book, raw)
	if validator.HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(prompts) != 1 {
		t.Fatalf("got %d prompts, want 1", len(prompts))
	}

	pt := prompts[0]
	if len(pt.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(pt.Segments))
	}
	if pt.Segments[1].Kind != model.SegmentSectionRef {
		t.Errorf("seg[1].Kind: got %v, want SegmentSectionRef", pt.Segments[1].Kind)
	}
	// #notitle: no title line, just laws
	expanded := pt.Segments[1].Expanded
	if expanded != "1.1 Run tests before proposing a change.\n\n1.2 Generated code must be reviewed." {
		t.Errorf("seg[1].Expanded: got %q", expanded)
	}
}

func TestExpand_PromptRef(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "base.prompt",
			Title:       "Base Prompt",
			Commentary:  "Base.",
			RawTemplate: "Base rules: {{ref:engineering.coding.run-tests}}",
			Source:      model.SourceRef{Path: "base.md"},
		},
		{
			ID:          "composed.prompt",
			Title:       "Composed Prompt",
			Commentary:  "Composed.",
			RawTemplate: "Start.\n{{ref:base.prompt}}\nEnd.",
			Source:      model.SourceRef{Path: "composed.md"},
		},
	}

	prompts, backlinks, diags := Expand(book, raw)
	if validator.HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(prompts) != 2 {
		t.Fatalf("got %d prompts, want 2", len(prompts))
	}

	// The composed prompt should include the base prompt's expanded template
	composed := prompts[1]
	if composed.ID != "composed.prompt" {
		t.Errorf("ID: got %q", composed.ID)
	}
	if !contains(composed.Template, "Base rules: 1.1 Run tests before proposing a change.") {
		t.Errorf("composed Template must contain expanded base, got %q", composed.Template)
	}

	// Transitive backlinks: engineering.coding.run-tests should be backlinked to both prompts
	if bl, ok := backlinks["engineering.coding.run-tests"]; !ok {
		t.Errorf("expected backlinks for engineering.coding.run-tests")
	} else {
		if len(bl) != 2 {
			t.Errorf("expected 2 backlinks, got %d: %v", len(bl), bl)
		}
	}
}

func TestExpand_DanglingReference(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "test.prompt",
			Title:       "Test",
			Commentary:  "Test.",
			RawTemplate: "{{ref:nonexistent.thing}}",
			Source:      model.SourceRef{Path: "test.md"},
		},
	}

	_, _, diags := Expand(book, raw)
	if !validator.HasErrors(diags) {
		t.Fatal("expected error for dangling reference")
	}
	found := false
	for _, d := range diags {
		if d.Code == "dangling-prompt-reference" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangling-prompt-reference diagnostic, got %v", diags)
	}
}

func TestExpand_CircularReference(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "a.prompt",
			Title:       "A",
			Commentary:  "A.",
			RawTemplate: "{{ref:b.prompt}}",
			Source:      model.SourceRef{Path: "a.md"},
		},
		{
			ID:          "b.prompt",
			Title:       "B",
			Commentary:  "B.",
			RawTemplate: "{{ref:a.prompt}}",
			Source:      model.SourceRef{Path: "b.md"},
		},
	}

	_, _, diags := Expand(book, raw)
	if !validator.HasErrors(diags) {
		t.Fatal("expected error for circular reference")
	}
	found := false
	for _, d := range diags {
		if d.Code == "circular-prompt-reference" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected circular-prompt-reference diagnostic, got %v", diags)
	}
}

func TestExpand_VarsExtracted(t *testing.T) {
	book := bookFixture()
	raw := []parser.ParsedPrompt{
		{
			ID:          "test.prompt",
			Title:       "Test",
			Commentary:  "Test.",
			RawTemplate: "Review in {{repo}} by {{author}}.\n{{ref:engineering.coding}}",
			Source:      model.SourceRef{Path: "test.md"},
		},
	}

	prompts, _, diags := Expand(book, raw)
	if validator.HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(prompts) != 1 {
		t.Fatalf("got %d prompts, want 1", len(prompts))
	}

	pt := prompts[0]
	if len(pt.Vars) != 2 {
		t.Fatalf("got %d vars, want 2: %v", len(pt.Vars), pt.Vars)
	}
	if pt.Vars[0] != "author" || pt.Vars[1] != "repo" {
		t.Errorf("vars: got %v, want [author repo]", pt.Vars)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
