package resolver

import (
	"errors"
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

func testBook() model.Lawbook {
	return model.Lawbook{
		Sections: []model.Section{
			{
				ID:     "engineering.security",
				Number: "1",
				Title:  "Security",
				Laws: []model.Law{
					{Number: "1.1", Index: 1, Text: "No secrets in SCM.", Slug: "no-secrets-in-scm", SectionID: "engineering.security"},
					{Number: "1.2", Index: 2, Text: "Use strong passwords.", Slug: "strong-passwords", SectionID: "engineering.security"},
				},
			},
			{
				ID:     "engineering.coding",
				Number: "2",
				Title:  "Coding",
				Laws: []model.Law{
					{Number: "2.1", Index: 1, Text: "Write tests.", Slug: "write-tests", SectionID: "engineering.coding"},
					{Number: "2.2", Index: 2, Text: "No secrets.", Slug: "no-secrets-in-scm", SectionID: "engineering.coding"},
				},
			},
			{
				ID:     "engineering.security.secrets",
				Number: "1.1",
				Title:  "Secrets",
				Laws: []model.Law{
					{Number: "1.1.1", Index: 1, Text: "Rotate keys.", Slug: "rotate-keys", SectionID: "engineering.security.secrets"},
				},
			},
		},
	}
}

func TestResolve_ExactSectionID(t *testing.T) {
	book := testBook()
	r, err := Resolve(book, "engineering.security")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindSection {
		t.Fatalf("expected KindSection, got %v", r.Kind)
	}
	if r.Section.ID != "engineering.security" {
		t.Errorf("wrong section: %s", r.Section.ID)
	}
}

func TestResolve_FullyQualifiedLawIdentity(t *testing.T) {
	book := testBook()
	r, err := Resolve(book, "engineering.security.no-secrets-in-scm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindLaw {
		t.Fatalf("expected KindLaw, got %v", r.Kind)
	}
	if r.Law.Number != "1.1" {
		t.Errorf("wrong law number: %s", r.Law.Number)
	}
}

func TestResolve_CitationNumber(t *testing.T) {
	book := testBook()
	r, err := Resolve(book, "2.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindLaw {
		t.Fatalf("expected KindLaw, got %v", r.Kind)
	}
	if r.Law.Text != "Write tests." {
		t.Errorf("wrong law: %s", r.Law.Text)
	}
}

func TestResolve_SectionNumber(t *testing.T) {
	book := testBook()
	r, err := Resolve(book, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindSection {
		t.Fatalf("expected KindSection, got %v", r.Kind)
	}
	if r.Section.ID != "engineering.security" {
		t.Errorf("wrong section: %s", r.Section.ID)
	}
}

func TestResolve_BareSlug_Unambiguous(t *testing.T) {
	book := testBook()
	// "write-tests" only exists in one section.
	r, err := Resolve(book, "write-tests")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindLaw {
		t.Fatalf("expected KindLaw, got %v", r.Kind)
	}
	if r.Law.SectionID != "engineering.coding" {
		t.Errorf("wrong section: %s", r.Law.SectionID)
	}
}

func TestResolve_BareSlug_Ambiguous(t *testing.T) {
	book := testBook()
	// "no-secrets-in-scm" exists in two sections.
	_, err := Resolve(book, "no-secrets-in-scm")
	if err == nil {
		t.Fatal("expected error for ambiguous bare slug")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_FullyQualifiedAmbiguousSlug(t *testing.T) {
	book := testBook()
	// The fully-qualified form should resolve even when the bare form is ambiguous.
	r, err := Resolve(book, "engineering.coding.no-secrets-in-scm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != KindLaw {
		t.Fatalf("expected KindLaw, got %v", r.Kind)
	}
	if r.Law.SectionID != "engineering.coding" {
		t.Errorf("wrong section: %s", r.Law.SectionID)
	}
}

func TestResolve_NotFound(t *testing.T) {
	book := testBook()
	_, err := Resolve(book, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent token")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_SectionIDPrecedenceOverLawIdentity(t *testing.T) {
	// If a section ID happens to be the same as a law's FQ identity,
	// the section ID match wins (highest precedence).
	book := model.Lawbook{
		Sections: []model.Section{
			{
				ID:     "a.b.c",
				Number: "1",
				Title:  "ABC",
				Laws: []model.Law{
					{Number: "1.1", Index: 1, Text: "Some law.", Slug: "c", SectionID: "a.b"},
				},
			},
		},
	}
	r, err := Resolve(book, "a.b.c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should resolve to the section, not the law (even though a.b.c is
	// also a valid FQ law identity a.b + slug c).
	if r.Kind != KindSection {
		t.Errorf("expected KindSection (precedence), got %v", r.Kind)
	}
}

func TestResolveLaw_BySlug(t *testing.T) {
	book := testBook()
	law, err := ResolveLaw(book, "strong-passwords")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if law.Number != "1.2" {
		t.Errorf("wrong law: %s", law.Number)
	}
}

func TestResolveLaw_ByNumber(t *testing.T) {
	book := testBook()
	law, err := ResolveLaw(book, "2.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if law.Text != "Write tests." {
		t.Errorf("wrong law: %s", law.Text)
	}
}

func TestAnchorFor_LawWithSlug(t *testing.T) {
	law := model.Law{Number: "1.1", Slug: "no-secrets-in-scm", SectionID: "engineering.security"}
	r := Resolved{Kind: KindLaw, Law: law}
	if got := AnchorFor(r); got != "engineering.security.no-secrets-in-scm" {
		t.Errorf("got %q, want %q", got, "engineering.security.no-secrets-in-scm")
	}
}

func TestAnchorFor_LawWithoutSlug(t *testing.T) {
	law := model.Law{Number: "1.1", SectionID: "engineering.security"}
	r := Resolved{Kind: KindLaw, Law: law}
	if got := AnchorFor(r); got != "1.1" {
		t.Errorf("got %q, want %q", got, "1.1")
	}
}

func TestAnchorFor_Section(t *testing.T) {
	sec := model.Section{ID: "engineering.security"}
	r := Resolved{Kind: KindSection, Section: sec}
	if got := AnchorFor(r); got != "engineering.security" {
		t.Errorf("got %q, want %q", got, "engineering.security")
	}
}
