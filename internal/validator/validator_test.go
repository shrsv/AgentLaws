package validator

import (
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

func law(sectionID string, number string, text string) model.Section {
	return model.Section{
		ID:     sectionID,
		Number: number,
		Laws:   []model.Law{{Number: number, Text: text, SectionID: sectionID}},
	}
}

func codes(diags []Diagnostic) map[string]bool {
	out := map[string]bool{}
	for _, d := range diags {
		out[d.Code] = true
	}
	return out
}

func TestValidate_UnfencedJSON_LawSingleTick(t *testing.T) {
	diags := Validate([]model.Section{law("a.b", "1.1", `Must match: `+"`json { \"decision\": \"approve\" }`"+`.`)})
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
	diags := Validate([]model.Section{s})
	if !codes(diags)["unfenced-json"] {
		t.Fatalf("expected unfenced-json warning for commentary, got %+v", diags)
	}
}

func TestValidate_NoUnfencedJSON_FencedBlockOK(t *testing.T) {
	text := "Must match:\n" +
		"   ```json\n" +
		"   {\"decision\": \"approve\"}\n" +
		"   ```\n"
	diags := Validate([]model.Section{law("a.b", "1.1", text)})
	if codes(diags)["unfenced-json"] {
		t.Fatalf("properly fenced json must not be flagged: %+v", diags)
	}
}

func TestValidate_NoUnfencedJSON_LoneWord(t *testing.T) {
	// `json` alone is a legitimate inline mention of the word, not a block.
	diags := Validate([]model.Section{law("a.b", "1.1", "Prefer `json` output.")})
	if codes(diags)["unfenced-json"] {
		t.Fatalf("lone `json` word must not be flagged: %+v", diags)
	}
}

func TestValidate_UnfencedJSON_UnclosedOpenerLine(t *testing.T) {
	text := "Shape:\n`json\n{\"a\": 1}\n`"
	diags := Validate([]model.Section{law("a.b", "1.1", text)})
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
		diags := Validate([]model.Section{law("a.b", "1.1", text)})
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
		diags := Validate([]model.Section{law("a.b", "1.1", text)})
		if codes(diags)["unfenced-json"] {
			t.Errorf("must not flag %q, got %+v", text, diags)
		}
	}
}
