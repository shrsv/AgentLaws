// Package lawedit locates the `<!-- alaws:laws -->` region in a section
// file and edits its numbered list without disturbing surrounding Markdown.
// It backs `alaws law add`/`alaws law remove` (PLAN1 §32).
package lawedit

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const lawsMarker = "<!-- alaws:laws -->"

var lawLineRe = regexp.MustCompile(`^\s*(\d+)\.\s+(.*)$`)

// splitAtLawsMarker returns everything up to and including the laws
// marker line (unchanged verbatim) and the existing clause texts found
// after it.
func splitAtLawsMarker(content string) (header string, clauses []string, err error) {
	lines := strings.Split(content, "\n")
	markerIdx := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == lawsMarker {
			markerIdx = i
			break
		}
	}
	if markerIdx == -1 {
		return "", nil, fmt.Errorf("lawedit: %s marker not found", lawsMarker)
	}

	header = strings.Join(lines[:markerIdx+1], "\n")

	var cur *string
	for i := markerIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if m := lawLineRe.FindStringSubmatch(line); m != nil {
			clauses = append(clauses, strings.TrimSpace(m[2]))
			cur = &clauses[len(clauses)-1]
			continue
		}
		if cur != nil && strings.TrimSpace(line) != "" {
			*cur = *cur + " " + strings.TrimSpace(line)
		}
	}
	return header, clauses, nil
}

func writeClauses(path, header string, clauses []string) error {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i, c := range clauses {
		b.WriteString("\n")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		b.WriteString(c)
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// Add appends a new numbered clause to the laws region of the section file
// at path. If after > 0, the clause is inserted immediately after that
// existing clause number instead of at the end.
func Add(path string, text string, after int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header, clauses, err := splitAtLawsMarker(string(data))
	if err != nil {
		return err
	}

	idx := len(clauses)
	if after > 0 && after <= len(clauses) {
		idx = after
	}
	clauses = append(clauses[:idx:idx], append([]string{text}, clauses[idx:]...)...)

	return writeClauses(path, header, clauses)
}

// Remove deletes the numbered clause `number` from the laws region of the
// section file at path, renumbering subsequent clauses.
func Remove(path string, number int, force bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header, clauses, err := splitAtLawsMarker(string(data))
	if err != nil {
		return err
	}
	if number < 1 || number > len(clauses) {
		return fmt.Errorf("lawedit: clause %d does not exist (section has %d)", number, len(clauses))
	}

	clauses = append(clauses[:number-1], clauses[number:]...)
	return writeClauses(path, header, clauses)
}
