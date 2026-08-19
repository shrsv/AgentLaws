package validator

import (
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

func law(sectionID string, number string, text string) model.Section {
	return model.Section{
		ID:     sectionID,
		Number: number,
		Laws:   []model.Law{{Number: number, Text: text, SectionID: sectionID, Slug: "test-law"}},
	}
}

func book(sections ...model.Section) model.Lawbook {
	return model.Lawbook{Sections: sections}
}

func codes(diags []Diagnostic) map[string]bool {
	out := map[string]bool{}
	for _, d := range diags {
		out[d.Code] = true
	}
	return out
}

func TestValidate_UnfencedJSON_LawSingleTick(t *testing.T) {
	diags := Validate(book(law("a.b", "1.1", `Must match: `+"`json { \"decision\": \"approve\" }`"+`.`)))
	if !codes(diags)["unfenced-json"] {
		t.Fatalf("expected unfenced-json warning, got %+v", diags)
	}
	for _, d := range diags {
		if d.Code == "unfenced-json" && d.Severity != SeverityWarning {
			t.Errorf("unfenced-json must be a warning, got %v", d.Severity)
		}
	}
}

func TestValidate_UnfencedJSON_Commentary(t *testing.T) {
	s := law("a.b", "1.1", "Fine.")
	s.Commentary = "Shape: `json { \"a\": 1 }`."
	diags := Validate(book(s))
	if !codes(diags)["unfenced-json"] {
		t.Fatalf("expected unfenced-json warning for commentary, got %+v", diags)
	}
}

func TestValidate_NoUnfencedJSON_FencedBlockOK(t *testing.T) {
	text := "Must match:\n" +
		"   ```json\n" +
		"   {\"decision\": \"approve\"}\n" +
		"   ```\n"
	diags := Validate(book(law("a.b", "1.1", text)))
	if codes(diags)["unfenced-json"] {
		t.Fatalf("properly fenced json must not be flagged: %+v", diags)
	}
}

func TestValidate_NoUnfencedJSON_LoneWord(t *testing.T) {
	// `json` alone is a legitimate inline mention of the word, not a block.
	diags := Validate(book(law("a.b", "1.1", "Prefer `json` output.")))
	if codes(diags)["unfenced-json"] {
		t.Fatalf("lone `json` word must not be flagged: %+v", diags)
	}
}

func TestValidate_UnfencedJSON_UnclosedOpenerLine(t *testing.T) {
	text := "Shape:\n`json\n{\"a\": 1}\n`"
	diags := Validate(book(law("a.b", "1.1", text)))
	if !codes(diags)["unfenced-json"] {
		t.Fatalf("expected unfenced-json warning for unclosed `json opener, got %+v", diags)
	}
}

func TestValidate_UnfencedJSON_JsonishObject(t *testing.T) {
	for _, text := range []string{
		"Value: `{\"a\": 1}`.",
		"Value: `{a: 1}`.",
		"Value: `[{\"a\": 1}]`.",
	} {
		diags := Validate(book(law("a.b", "1.1", text)))
		if !codes(diags)["unfenced-json"] {
			t.Errorf("expected unfenced-json for %q, got %+v", text, diags)
		}
	}
}

func TestValidate_NoUnfencedJSON_NotJsonish(t *testing.T) {
	for _, text := range []string{
		"See `[foo]: https://example.com`.",
		"Map `{a}: {b}`.",
		"Use `key`.",
	} {
		diags := Validate(book(law("a.b", "1.1", text)))
		if codes(diags)["unfenced-json"] {
			t.Errorf("must not flag %q, got %+v", text, diags)
		}
	}
}

// --- Linking diagnostics ---

func TestValidate_MissingSlug(t *testing.T) {
	s := model.Section{
		ID:     "a.b",
		Number: "1",
		Laws: []model.Law{
			{Number: "1.1", Text: "Some law.", SectionID: "a.b", Slug: ""},
		},
	}
	diags := Validate(book(s))
	if !codes(diags)["missing-slug"] {
		t.Fatalf("expected missing-slug, got %+v", diags)
	}
	for _, d := range diags {
		if d.Code == "missing-slug" && d.Severity != SeverityError {
			t.Errorf("missing-slug must be error, got %v", d.Severity)
		}
	}
}

func TestValidate_InvalidSlug(t *testing.T) {
	s := model.Section{
		ID:     "a.b",
		Number: "1",
		Laws: []model.Law{
			{Number: "1.1", Text: "Some law.", SectionID: "a.b", Slug: "Bad_Slug!"},
		},
	}
	diags := Validate(book(s))
	if !codes(diags)["invalid-slug"] {
		t.Fatalf("expected invalid-slug, got %+v", diags)
	}
}

func TestValidate_DuplicateSlug(t *testing.T) {
	s := model.Section{
		ID:     "a.b",
		Number: "1",
		Laws: []model.Law{
			{Number: "1.1", Text: "Law one.", SectionID: "a.b", Slug: "same-slug", Source: model.SourceRef{Path: "a.md", LineStart: 1}},
			{Number: "1.2", Text: "Law two.", SectionID: "a.b", Slug: "same-slug", Source: model.SourceRef{Path: "a.md", LineStart: 2}},
		},
	}
	diags := Validate(book(s))
	if !codes(diags)["duplicate-slug"] {
		t.Fatalf("expected duplicate-slug, got %+v", diags)
	}
}

func TestValidate_AmbiguousIdentity(t *testing.T) {
	// A law's FQ identity matches another section's ID.
	s1 := model.Section{
		ID:     "a.b",
		Number: "1",
		Laws: []model.Law{
			{Number: "1.1", Text: "Some law.", SectionID: "a.b", Slug: "c"},
		},
	}
	s2 := model.Section{
		ID:     "a.b.c",
		Number: "2",
		Laws:  []model.Law{},
	}
	diags := Validate(book(s1, s2))
	if !codes(diags)["ambiguous-identity"] {
		t.Fatalf("expected ambiguous-identity, got %+v", diags)
	}
}

func TestValidate_DanglingReference(t *testing.T) {
	s := model.Section{
		ID:         "a.b",
		Number:     "1",
		Commentary: "See [this](alaws:nonexistent) for details.",
		Laws: []model.Law{
			{Number: "1.1", Text: "Some law.", SectionID: "a.b", Slug: "some-law"},
		},
	}
	diags := Validate(book(s))
	if !codes(diags)["dangling-reference"] {
		t.Fatalf("expected dangling-reference, got %+v", diags)
	}
}

func TestValidate_ValidDanglingReference(t *testing.T) {
	// A valid alaws: link should not produce a dangling-reference.
	s := model.Section{
		ID:         "a.b",
		Number:     "1",
		Commentary: "See [this](alaws:a.b.some-law) for details.",
		Laws: []model.Law{
			{Number: "1.1", Text: "Some law.", SectionID: "a.b", Slug: "some-law"},
		},
	}
	diags := Validate(book(s))
	if codes(diags)["dangling-reference"] {
		t.Fatalf("valid alaws: link must not be flagged: %+v", diags)
	}
}

func TestValidate_AllDiagnosticsClean_WithSlugs(t *testing.T) {
	// A properly formed section with slugs should have no linking diagnostics.
	s := model.Section{
		ID:     "a.b",
		Number: "1",
		Laws: []model.Law{
			{Number: "1.1", Text: "Law one.", SectionID: "a.b", Slug: "law-one"},
			{Number: "1.2", Text: "Law two.", SectionID: "a.b", Slug: "law-two"},
		},
	}
	diags := Validate(book(s))
	for _, d := range diags {
		switch d.Code {
		case "missing-slug", "invalid-slug", "duplicate-slug", "ambiguous-identity", "dangling-reference":
			t.Errorf("unexpected diagnostic %s: %s", d.Code, d.Message)
		}
	}
}
