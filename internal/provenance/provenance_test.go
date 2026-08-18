package provenance

import (
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

func TestHashLawbookDeterministic(t *testing.T) {
	book := model.Lawbook{
		Metadata: model.LawbookMetadata{Title: "Test", Ordering: []string{"a.md"}},
		Sections: []model.Section{
			{
				ID:         "sec1",
				Number:     "1",
				Title:      "Section One",
				Level:      1,
				Commentary: "Some commentary",
				Laws: []model.Law{
					{Number: "1.1", Index: 1, Text: "First law", SectionID: "sec1"},
					{Number: "1.2", Index: 2, Text: "Second law", SectionID: "sec1"},
				},
			},
		},
	}

	h1, err := HashLawbook(book)
	if err != nil {
		t.Fatalf("HashLawbook first call: %v", err)
	}
	h2, err := HashLawbook(book)
	if err != nil {
		t.Fatalf("HashLawbook second call: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("non-deterministic hash: %s != %s", h1, h2)
	}
	if h1 == "" {
		t.Fatal("empty hash")
	}
}

func TestHashLawbookExcludesProvenance(t *testing.T) {
	book := model.Lawbook{
		Metadata: model.LawbookMetadata{Title: "Test"},
		Sections: []model.Section{{ID: "s", Number: "1", Title: "S", Level: 1}},
	}
	book.Provenance = model.Provenance{Revision: "abc123", CompiledAt: "2025-01-01T00:00:00Z"}
	h1, err := HashLawbook(book)
	if err != nil {
		t.Fatal(err)
	}

	book.Provenance = model.Provenance{Revision: "def456", CompiledAt: "2025-06-15T12:00:00Z"}
	h2, err := HashLawbook(book)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("provenance change affected content hash: %s != %s", h1, h2)
	}
}

func TestHashSectionMatches(t *testing.T) {
	s := model.Section{
		ID: "test", Number: "1", Title: "Test", Level: 1,
		Laws: []model.Law{{Number: "1.1", Index: 1, Text: "law", SectionID: "test"}},
	}
	h, err := HashSection(s)
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Fatal("empty hash")
	}
}

func TestDiffDetectsAddedSection(t *testing.T) {
	old := model.Lawbook{Sections: []model.Section{{ID: "a", Number: "1", Title: "A", Level: 1}}}
	new := model.Lawbook{Sections: []model.Section{
		{ID: "a", Number: "1", Title: "A", Level: 1},
		{ID: "b", Number: "2", Title: "B", Level: 1},
	}}
	d := Diff(old, new)
	if len(d.AddedSections) != 1 || d.AddedSections[0] != "b" {
		t.Fatalf("expected added section b, got %v", d.AddedSections)
	}
	if len(d.RemovedSections) != 0 {
		t.Fatalf("expected no removed sections, got %v", d.RemovedSections)
	}
}

func TestDiffDetectsRemovedSection(t *testing.T) {
	old := model.Lawbook{Sections: []model.Section{
		{ID: "a", Number: "1", Title: "A", Level: 1},
		{ID: "b", Number: "2", Title: "B", Level: 1},
	}}
	new := model.Lawbook{Sections: []model.Section{{ID: "a", Number: "1", Title: "A", Level: 1}}}
	d := Diff(old, new)
	if len(d.RemovedSections) != 1 || d.RemovedSections[0] != "b" {
		t.Fatalf("expected removed section b, got %v", d.RemovedSections)
	}
}

func TestDiffDetectsModifiedLaw(t *testing.T) {
	old := model.Lawbook{Sections: []model.Section{{
		ID: "s", Number: "1", Title: "S", Level: 1,
		Laws: []model.Law{{Number: "1.1", Index: 1, Text: "old text", SectionID: "s"}},
	}}}
	new := model.Lawbook{Sections: []model.Section{{
		ID: "s", Number: "1", Title: "S", Level: 1,
		Laws: []model.Law{{Number: "1.1", Index: 1, Text: "new text", SectionID: "s"}},
	}}}
	d := Diff(old, new)
	if len(d.ModifiedLaws) != 1 {
		t.Fatalf("expected 1 modified law, got %d", len(d.ModifiedLaws))
	}
	if d.ModifiedLaws[0].OldText != "old text" || d.ModifiedLaws[0].NewText != "new text" {
		t.Fatalf("unexpected modification: %v", d.ModifiedLaws[0])
	}
}

func TestDiffIgnoresReorder(t *testing.T) {
	old := model.Lawbook{Sections: []model.Section{
		{ID: "a", Number: "1", Title: "A", Level: 1, Laws: []model.Law{{Number: "1.1", Index: 1, Text: "x", SectionID: "a"}}},
		{ID: "b", Number: "2", Title: "B", Level: 1, Laws: []model.Law{{Number: "2.1", Index: 1, Text: "y", SectionID: "b"}}},
	}}
	new := model.Lawbook{Sections: []model.Section{
		{ID: "b", Number: "1", Title: "B", Level: 1, Laws: []model.Law{{Number: "1.1", Index: 1, Text: "y", SectionID: "b"}}},
		{ID: "a", Number: "2", Title: "A", Level: 1, Laws: []model.Law{{Number: "2.1", Index: 1, Text: "x", SectionID: "a"}}},
	}}
	d := Diff(old, new)
	if len(d.AddedSections) != 0 || len(d.RemovedSections) != 0 {
		t.Fatalf("reorder reported as add/remove: added=%v removed=%v", d.AddedSections, d.RemovedSections)
	}
	if len(d.ModifiedLaws) != 0 {
		t.Fatalf("reorder reported as modified: %v", d.ModifiedLaws)
	}
}

func TestDiffDetectsAddedLaw(t *testing.T) {
	old := model.Lawbook{Sections: []model.Section{{
		ID: "s", Number: "1", Title: "S", Level: 1,
		Laws: []model.Law{{Number: "1.1", Index: 1, Text: "existing", SectionID: "s"}},
	}}}
	new := model.Lawbook{Sections: []model.Section{{
		ID: "s", Number: "1", Title: "S", Level: 1,
		Laws: []model.Law{
			{Number: "1.1", Index: 1, Text: "existing", SectionID: "s"},
			{Number: "1.2", Index: 2, Text: "new law", SectionID: "s"},
		},
	}}}
	d := Diff(old, new)
	if len(d.AddedLaws) != 1 || d.AddedLaws[0].Text != "new law" {
		t.Fatalf("expected added law 'new law', got %v", d.AddedLaws)
	}
}
