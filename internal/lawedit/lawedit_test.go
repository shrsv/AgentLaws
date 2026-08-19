package lawedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip_FencedLawWithOwnLineSlug(t *testing.T) {
	// This is the regression test for the pre-existing bug: a fenced,
	// multi-line law with an own-line {#slug} must survive Add + Remove
	// unchanged.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	original := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Some commentary.

<!-- alaws:laws -->

1. Run this check before merging:
   ` + "```bash" + `
   make test
   ` + "```" + `
   {#pre-merge-check}
2. Second law. {#second-law}
`
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	// Add a third law.
	if err := Add(path, "Third law.", "", 0); err != nil {
		t.Fatal(err)
	}

	// Remove the third law.
	if err := Remove(path, 3, true); err != nil {
		t.Fatal(err)
	}

	// Read back and compare.
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	// The first law's text must still contain the fenced block.
	if !strings.Contains(got, "```bash") {
		t.Error("fenced block was destroyed by Add+Remove round trip")
	}
	if !strings.Contains(got, "make test") {
		t.Error("fenced block content was destroyed")
	}
	// The slug must still be present.
	if !strings.Contains(got, "{#pre-merge-check}") {
		t.Error("own-line slug was destroyed by Add+Remove round trip")
	}
	if !strings.Contains(got, "{#second-law}") {
		t.Error("inline slug was destroyed by Add+Remove round trip")
	}
}

func TestAdd_AutoSlug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Commentary.

<!-- alaws:laws -->

`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Add(path, "Credentials must never be committed to source control.", "", 0); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	if !strings.Contains(got, "{#credentials-must-never-be") {
		t.Errorf("expected auto-generated slug, got:\n%s", got)
	}
}

func TestAdd_ExplicitSlug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Commentary.

<!-- alaws:laws -->

`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Add(path, "Some law text.", "my-custom-slug", 0); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	if !strings.Contains(got, "{#my-custom-slug}") {
		t.Errorf("expected explicit slug, got:\n%s", got)
	}
}

func TestSetSlug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Commentary.

<!-- alaws:laws -->

1. First law. {#old-slug}
2. Second law. {#other-slug}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SetSlug(path, "1", "new-slug"); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	if !strings.Contains(got, "{#new-slug}") {
		t.Errorf("expected new slug, got:\n%s", got)
	}
	if strings.Contains(got, "{#old-slug}") {
		t.Errorf("old slug should be gone, got:\n%s", got)
	}
}

func TestSetSlug_ByExistingSlug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Commentary.

<!-- alaws:laws -->

1. First law. {#find-me}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SetSlug(path, "find-me", "replaced"); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	if !strings.Contains(got, "{#replaced}") {
		t.Errorf("expected replaced slug, got:\n%s", got)
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		slug string
		want bool
	}{
		{"good-slug", true},
		{"abc123", true},
		{"a", true},
		{"no-secrets-in-scm", true},
		{"", false},
		{"Bad", false},
		{"has_underscore", false},
		{"has.dot", false},
		{"123-start", false},
		{"-start-dash", false},
	}
	for _, tt := range tests {
		if got := IsValidSlug(tt.slug); got != tt.want {
			t.Errorf("IsValidSlug(%q) = %v, want %v", tt.slug, got, tt.want)
		}
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		text     string
		existing []string
		want     string
	}{
		{"Credentials must never be committed.", nil, "credentials-must-never-be-committed"},
		{"Write tests for everything.", nil, "write-tests-for-everything"},
		{"Short.", nil, "short"},
		{"", nil, "law"},
		{"Same text.", []string{"same-text"}, "same-text-2"},
	}
	for _, tt := range tests {
		got := GenerateSlug(tt.text, tt.existing)
		if got != tt.want {
			t.Errorf("GenerateSlug(%q, %v) = %q, want %q", tt.text, tt.existing, got, tt.want)
		}
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	content := `---
title: Test
id: test.section
---

<!-- alaws:commentary -->

Commentary.

<!-- alaws:laws -->

1. First law. {#first}
2. Second law. {#second}
3. Third law. {#third}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Remove(path, 2, true); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	if strings.Contains(got, "Second law") {
		t.Error("second law should be removed")
	}
	if !strings.Contains(got, "First law") {
		t.Error("first law should remain")
	}
	if !strings.Contains(got, "Third law") {
		t.Error("third law should remain")
	}
}
