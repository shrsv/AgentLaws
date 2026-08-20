package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseLawLines_ProseFolds(t *testing.T) {
	src := "1. This is a long clause\n   wrapped across two lines.\n\n2. A second law.\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 2 {
		t.Fatalf("got %d laws, want 2", len(got))
	}
	want := "This is a long clause wrapped across two lines."
	if got[0].Text != want {
		t.Errorf("law 1: got %q, want %q", got[0].Text, want)
	}
}

func TestParseLawLines_JSONBlockPreservesNewlines(t *testing.T) {
	src := "1. The response must match this schema:\n" +
		"   ```json\n" +
		"   {\n" +
		"     \"a\": 1,\n" +
		"     \"b\": [1, 2]\n" +
		"   }\n" +
		"   ```\n" +
		"2. And another law.\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 2 {
		t.Fatalf("got %d laws, want 2", len(got))
	}
	want := "The response must match this schema:\n" +
		"   ```json\n" +
		"   {\n" +
		"     \"a\": 1,\n" +
		"     \"b\": [1, 2]\n" +
		"   }\n" +
		"   ```"
	if got[0].Text != want {
		t.Errorf("law 1:\n got %q\nwant %q", got[0].Text, want)
	}
	if got[0].Text == strings.ReplaceAll(got[0].Text, "\n", " ") {
		t.Error("law 1 must not be folded onto a single line")
	}
}

func TestParseLawLines_NumberedLineInsideFenceIsNotNewLaw(t *testing.T) {
	src := "1. Config:\n" +
		"   ```json\n" +
		"   {\"items\": [\n" +
		"     2. this line looks numbered but is JSON\n" +
		"   ]}\n" +
		"   ```\n" +
		"2. Next law.\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 2 {
		t.Fatalf("got %d laws, want 2 (line inside code fence must not start a new law)", len(got))
	}
	if !strings.Contains(got[0].Text, "2. this line looks numbered") {
		t.Errorf("law 1 must keep the in-fence numbered line, got %q", got[0].Text)
	}
}

func TestParseLawLines_FenceOnNumberedLine(t *testing.T) {
	src := "1. ```json\n" +
		"   {\"x\": 1}\n" +
		"   ```\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	want := "```json\n   {\"x\": 1}\n   ```"
	if got[0].Text != want {
		t.Errorf("law 1: got %q, want %q", got[0].Text, want)
	}
}

func TestParseLawLines_UnclosedFenceKeepsRemainder(t *testing.T) {
	src := "1. Schema:\n" +
		"   ```\n" +
		"   keep me\n" +
		"   not a law\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	want := "Schema:\n   ```\n   keep me\n   not a law"
	if got[0].Text != want {
		t.Errorf("law 1: got %q, want %q", got[0].Text, want)
	}
}

// --- Slug extraction tests ---

func TestParseLawLines_SlugInline(t *testing.T) {
	src := "1. Credentials must never be committed. {#no-secrets-in-scm}\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "no-secrets-in-scm" {
		t.Errorf("slug: got %q, want %q", got[0].Slug, "no-secrets-in-scm")
	}
	if got[0].Text != "Credentials must never be committed." {
		t.Errorf("text: got %q, want %q", got[0].Text, "Credentials must never be committed.")
	}
}

func TestParseLawLines_SlugOwnLineAfterFence(t *testing.T) {
	src := "1. Run this check:\n" +
		"   ```bash\n" +
		"   make test\n" +
		"   ```\n" +
		"   {#pre-merge-check}\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "pre-merge-check" {
		t.Errorf("slug: got %q, want %q", got[0].Slug, "pre-merge-check")
	}
	if !strings.Contains(got[0].Text, "```bash") {
		t.Errorf("text must contain fenced block, got %q", got[0].Text)
	}
	// The slug should be stripped from the text.
	if strings.Contains(got[0].Text, "{#pre-merge-check}") {
		t.Errorf("text must not contain slug attribute, got %q", got[0].Text)
	}
}

func TestParseLawLines_SlugNegativeCase(t *testing.T) {
	// A law whose text ends with something brace-shaped but not matching
	// the slug charset must be left untouched.
	src := "1. See the full stop {End}.\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "" {
		t.Errorf("slug should be empty, got %q", got[0].Slug)
	}
	if !strings.Contains(got[0].Text, "{End}") {
		t.Errorf("text must keep {End} intact, got %q", got[0].Text)
	}
}

func TestParseLawLines_SlugUppercaseNotMatched(t *testing.T) {
	// Uppercase in slug position must not be treated as a slug.
	src := "1. Some text. {#Not-A-Slug}\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "" {
		t.Errorf("slug should be empty for uppercase, got %q", got[0].Slug)
	}
	if !strings.Contains(got[0].Text, "{#Not-A-Slug}") {
		t.Errorf("text must keep the uppercase attribute, got %q", got[0].Text)
	}
}

func TestParseLawLines_SlugDigitStartNotMatched(t *testing.T) {
	// A slug starting with a digit must not be matched.
	src := "1. Some text. {#123bad}\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "" {
		t.Errorf("slug should be empty for digit-start, got %q", got[0].Slug)
	}
}

func TestParseLawLines_SlugMultipleOnOwnLine(t *testing.T) {
	// Multi-line law with slug on its own line (no fence).
	src := "1. This is a longer law\n   that spans lines.\n   {#multi-line-law}\n"
	got := ParseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	if got[0].Slug != "multi-line-law" {
		t.Errorf("slug: got %q, want %q", got[0].Slug, "multi-line-law")
	}
	if strings.Contains(got[0].Text, "{#multi-line-law}") {
		t.Errorf("text must not contain slug, got %q", got[0].Text)
	}
}

// --- ParsePromptTemplate tests ---

func TestParsePromptTemplate_Valid(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test-prompt.md"
	content := `---
title: Code Review Prompt
id: engineering.prompts.code-review
---

<!-- alaws:commentary -->

Used by the CI review bot.

<!-- alaws:promptTemplate -->

You are reviewing a PR in {{repo}}.

{{ref:engineering.coding}}
`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	pp, err := ParsePromptTemplate(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pp.ID != "engineering.prompts.code-review" {
		t.Errorf("ID: got %q, want %q", pp.ID, "engineering.prompts.code-review")
	}
	if pp.Title != "Code Review Prompt" {
		t.Errorf("Title: got %q, want %q", pp.Title, "Code Review Prompt")
	}
	if pp.Commentary != "Used by the CI review bot." {
		t.Errorf("Commentary: got %q", pp.Commentary)
	}
	if !strings.Contains(pp.RawTemplate, "{{ref:engineering.coding}}") {
		t.Errorf("RawTemplate must contain ref directive, got %q", pp.RawTemplate)
	}
	if !strings.Contains(pp.RawTemplate, "{{repo}}") {
		t.Errorf("RawTemplate must contain var placeholder, got %q", pp.RawTemplate)
	}
}

func TestParsePromptTemplate_MissingTitle(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.md"
	content := `---
id: some.id
---

<!-- alaws:commentary -->
Commentary here.

<!-- alaws:promptTemplate -->
Template body.
`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	_, err := ParsePromptTemplate(path)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "missing-title") {
		t.Errorf("expected missing-title error, got: %v", err)
	}
}

func TestParsePromptTemplate_MissingID(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.md"
	content := `---
title: Some Title
---

<!-- alaws:commentary -->
Commentary here.

<!-- alaws:promptTemplate -->
Template body.
`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	_, err := ParsePromptTemplate(path)
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	if !strings.Contains(err.Error(), "missing-id") {
		t.Errorf("expected missing-id error, got: %v", err)
	}
}

func TestParsePromptTemplate_MissingPromptTemplateMarker(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.md"
	content := `---
title: Some Title
id: some.id
---

<!-- alaws:commentary -->
Commentary here.
`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	_, err := ParsePromptTemplate(path)
	if err == nil {
		t.Fatal("expected error for missing promptTemplate marker")
	}
	if !strings.Contains(err.Error(), "missing-prompt-template") {
		t.Errorf("expected missing-prompt-template error, got: %v", err)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
