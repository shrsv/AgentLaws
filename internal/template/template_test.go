package template

import (
	"errors"
	"testing"
)

func TestRenderSubstitutes(t *testing.T) {
	got, err := Render("Agent {{agent_name}} must not touch {{repo}}.", map[string]string{
		"agent_name": "ci-bot",
		"repo":       "org/app",
	}, MissingError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Agent ci-bot must not touch org/app."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderEscapedBrace(t *testing.T) {
	got, err := Render(`literal \{{not a var}} stays`, nil, MissingKeepPlaceholder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "literal {{not a var}} stays"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderMissingErrorPolicy(t *testing.T) {
	_, err := Render("{{unset}}", nil, MissingError)
	if err == nil {
		t.Fatal("expected error for missing variable, got nil")
	}
	var mv *MissingVariableError
	if !errors.As(err, &mv) {
		t.Fatalf("expected MissingVariableError, got %T: %v", err, err)
	}
	if mv.Name != "unset" {
		t.Errorf("got %q, want %q", mv.Name, "unset")
	}
}

func TestRenderMissingKeepPolicy(t *testing.T) {
	got, err := Render("before {{unset}} after", nil, MissingKeepPlaceholder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "before {{unset}} after" {
		t.Errorf("got %q", got)
	}
}

func TestRenderMissingEmptyPolicy(t *testing.T) {
	got, err := Render("before {{unset}} after", nil, MissingEmpty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "before  after" {
		t.Errorf("got %q", got)
	}
}

func TestValidateSyntaxOK(t *testing.T) {
	if err := ValidateSyntax("law about {{agent_name}} and {{env.region}}"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSyntaxUnterminated(t *testing.T) {
	if err := ValidateSyntax("law about {{agent_name"); err == nil {
		t.Error("expected error for unterminated placeholder")
	}
}

func TestValidateSyntaxInvalidIdentifier(t *testing.T) {
	cases := []string{"{{}}", "{{1abc}}", "{{has space}}", "{{bad-char}}"}
	for _, c := range cases {
		if err := ValidateSyntax(c); err == nil {
			t.Errorf("expected error for %q, got nil", c)
		}
	}
}

func TestValidateSyntaxEscaped(t *testing.T) {
	if err := ValidateSyntax(`\{{ not a placeholder }}`); err != nil {
		t.Errorf("unexpected error for escaped brace: %v", err)
	}
}
