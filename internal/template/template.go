// Package template implements variable substitution for law and commentary
// text, as designed in docs/PLAN1.md §17a.
//
// Placeholders are plain `{{identifier}}` substitution only - no
// conditionals, loops, or function calls. Resolution happens at the
// extraction/render boundary and never mutates the canonical Lawbook IR, so
// compilation and signing stay independent of variable values.
package template

import (
	"fmt"
	"sort"
	"strings"
)

// MissingPolicy controls Render's behavior when a placeholder has no
// corresponding entry in vars.
type MissingPolicy int

const (
	// MissingError fails the render if any placeholder is unresolved. Default.
	MissingError MissingPolicy = iota
	// MissingKeepPlaceholder leaves `{{x}}` untouched in the output.
	MissingKeepPlaceholder
	// MissingEmpty substitutes the empty string.
	MissingEmpty
)

func isIdentByte(b byte, first bool) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b == '_':
		return true
	case b >= '0' && b <= '9':
		return !first
	case b == '.':
		return !first
	default:
		return false
	}
}

// ValidateSyntax checks that every `{{...}}` placeholder in text is
// well-formed: balanced braces and a valid identifier. It does not require
// the identifier to have a value. This is what the compiler runs at compile
// time (diagnostic code invalid-template, §19); it never inspects vars.
func ValidateSyntax(text string) error {
	i := 0
	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) && text[i+1] == '{' {
			i += 2
			continue
		}
		if text[i] == '{' && i+1 < len(text) && text[i+1] == '{' {
			end := strings.Index(text[i:], "}}")
			if end == -1 {
				return fmt.Errorf("invalid-template: unterminated placeholder at offset %d", i)
			}
			ident := text[i+2 : i+end]
			if err := validateIdentifier(ident); err != nil {
				return fmt.Errorf("invalid-template: %w (at offset %d)", err, i)
			}
			i += end + 2
			continue
		}
		i++
	}
	return nil
}

func validateIdentifier(ident string) error {
	if ident == "" {
		return fmt.Errorf("empty placeholder %q", "{{}}")
	}
	for j := 0; j < len(ident); j++ {
		if !isIdentByte(ident[j], j == 0) {
			return fmt.Errorf("invalid identifier %q", ident)
		}
	}
	return nil
}

// MissingVariableError is returned by Render when policy is MissingError and
// a placeholder has no entry in vars.
type MissingVariableError struct {
	Name string
}

func (e *MissingVariableError) Error() string {
	return fmt.Sprintf("template: missing value for variable %q", e.Name)
}

// Render substitutes `{{identifier}}` placeholders in text with values from
// vars. `\{{` is treated as a literal `{{`. See docs/PLAN1.md §17a.
func Render(text string, vars map[string]string, policy MissingPolicy) (string, error) {
	var out strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) && text[i+1] == '{' {
			out.WriteByte('{')
			i += 2
			continue
		}
		if text[i] == '{' && i+1 < len(text) && text[i+1] == '{' {
			end := strings.Index(text[i:], "}}")
			if end == -1 {
				return "", fmt.Errorf("template: unterminated placeholder at offset %d", i)
			}
			ident := text[i+2 : i+end]
			if err := validateIdentifier(ident); err != nil {
				return "", fmt.Errorf("template: %w (at offset %d)", err, i)
			}
			val, ok := vars[ident]
			if !ok {
				switch policy {
				case MissingKeepPlaceholder:
					out.WriteString(text[i : i+end+2])
				case MissingEmpty:
					// write nothing
				default:
					return "", &MissingVariableError{Name: ident}
				}
			} else {
				out.WriteString(val)
			}
			i += end + 2
			continue
		}
		out.WriteByte(text[i])
		i++
	}
	return out.String(), nil
}

// Vars returns the sorted, deduplicated list of {{identifier}} names found
// in text. It reuses ValidateSyntax's walk logic. Escaped `\{{` sequences
// are skipped. This is used by the prompt expansion pipeline to compute
// PromptTemplate.Vars.
func Vars(text string) []string {
	seen := map[string]bool{}
	i := 0
	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) && text[i+1] == '{' {
			i += 2
			continue
		}
		if text[i] == '{' && i+1 < len(text) && text[i+1] == '{' {
			end := strings.Index(text[i:], "}}")
			if end == -1 {
				break
			}
			ident := text[i+2 : i+end]
			if validateIdentifier(ident) == nil {
				seen[ident] = true
			}
			i += end + 2
			continue
		}
		i++
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
