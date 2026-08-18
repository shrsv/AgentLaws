package parser

import (
	"strings"
	"testing"
)

func TestParseLawLines_ProseFolds(t *testing.T) {
	src := "1. This is a long clause\n   wrapped across two lines.\n\n2. A second law.\n"
	got := parseLawLines(strings.Split(src, "\n"), 0)

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
	got := parseLawLines(strings.Split(src, "\n"), 0)

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
	got := parseLawLines(strings.Split(src, "\n"), 0)

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
	got := parseLawLines(strings.Split(src, "\n"), 0)

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
	got := parseLawLines(strings.Split(src, "\n"), 0)

	if len(got) != 1 {
		t.Fatalf("got %d laws, want 1", len(got))
	}
	want := "Schema:\n   ```\n   keep me\n   not a law"
	if got[0].Text != want {
		t.Errorf("law 1: got %q, want %q", got[0].Text, want)
	}
}
