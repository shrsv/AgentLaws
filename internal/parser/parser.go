// Package parser parses alaws.toml and section Markdown files into raw,
// unvalidated data ready for the compiler. See docs/PLAN1.md §6-§11.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/athreyac4/agentlaws/internal/model"
)

const (
	commentaryMarker = "<!-- alaws:commentary -->"
	lawsMarker       = "<!-- alaws:laws -->"
)

// ResolveLevel returns a section's presentation level: override, if the
// section's frontmatter set one explicitly, otherwise 1 plus the number of
// path separators in entryPath (its ordering entry, relative to the
// lawbook root).
//
// Folder names carry no meaning (PLAN1 §2.1), but nesting *depth* is a
// convenient default for level: a file one directory down from the book
// root is naturally a level-2 section, two down is level 3, and so on,
// without an author needing to write `level:` in every file. Frontmatter
// `level:` remains available to express the exception - a section whose
// intended presentation depth doesn't match where its file happens to
// live.
func ResolveLevel(entryPath string, override *int) int {
	if override != nil {
		return *override
	}
	return 1 + strings.Count(filepath.ToSlash(entryPath), "/")
}

var lawLineRe = regexp.MustCompile(`^\s*(\d+)\.\s+(.*)$`)

// RawLaw is one numbered clause as found in the laws region, before
// canonical numbering is assigned by the compiler.
type RawLaw struct {
	Text      string
	LineStart int
	LineEnd   int
}

// ParsedSection is the raw result of parsing one section file, before
// validation or numbering.
type ParsedSection struct {
	ID         string
	Title      string
	Level      *int // nil if not set in frontmatter
	Commentary string
	RawLaws    []RawLaw
	Source     model.SourceRef
}

type tomlConfig struct {
	Title    string   `toml:"title"`
	Ordering []string `toml:"ordering"`
}

// ParseLawbookConfig parses an alaws.toml file.
func ParseLawbookConfig(path string) (model.LawbookMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.LawbookMetadata{}, err
	}
	var cfg tomlConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return model.LawbookMetadata{}, fmt.Errorf("invalid-metadata: %s: %w", path, err)
	}
	return model.LawbookMetadata{Title: cfg.Title, Ordering: cfg.Ordering}, nil
}

type frontmatter struct {
	Title string `yaml:"title"`
	ID    string `yaml:"id"`
	Level *int   `yaml:"level"`
}

// ParseSection parses one Markdown section file into frontmatter,
// commentary, and raw law clauses.
func ParseSection(path string) (ParsedSection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParsedSection{}, err
	}
	lines := strings.Split(string(data), "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ParsedSection{}, fmt.Errorf("invalid-metadata: %s: file must start with YAML frontmatter delimited by ---", path)
	}
	fmEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fmEnd = i
			break
		}
	}
	if fmEnd == -1 {
		return ParsedSection{}, fmt.Errorf("invalid-metadata: %s: unterminated frontmatter", path)
	}

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:fmEnd], "\n")), &fm); err != nil {
		return ParsedSection{}, fmt.Errorf("invalid-metadata: %s: %w", path, err)
	}
	if fm.Title == "" {
		return ParsedSection{}, fmt.Errorf("missing-title: %s", path)
	}
	if fm.ID == "" {
		return ParsedSection{}, fmt.Errorf("missing-id: %s", path)
	}

	commentaryLine, lawsLine := -1, -1
	for i := fmEnd + 1; i < len(lines); i++ {
		switch strings.TrimSpace(lines[i]) {
		case commentaryMarker:
			if commentaryLine == -1 {
				commentaryLine = i
			}
		case lawsMarker:
			if lawsLine == -1 {
				lawsLine = i
			}
		}
	}
	if commentaryLine == -1 {
		return ParsedSection{}, fmt.Errorf("missing-commentary: %s: missing %s marker", path, commentaryMarker)
	}
	if lawsLine == -1 {
		return ParsedSection{}, fmt.Errorf("missing-laws: %s: missing %s marker", path, lawsMarker)
	}
	if lawsLine < commentaryLine {
		return ParsedSection{}, fmt.Errorf("invalid-metadata: %s: %s must appear before %s", path, commentaryMarker, lawsMarker)
	}

	commentary := strings.TrimSpace(strings.Join(lines[commentaryLine+1:lawsLine], "\n"))
	rawLaws := parseLawLines(lines[lawsLine+1:], lawsLine+1)

	return ParsedSection{
		ID:         fm.ID,
		Title:      fm.Title,
		Level:      fm.Level,
		Commentary: commentary,
		RawLaws:    rawLaws,
		Source:     model.SourceRef{Path: path, LineStart: 1, LineEnd: len(lines)},
	}, nil
}

// parseLawLines extracts numbered clauses from the laws region. A clause
// starts with "<N>. " at the beginning of a line; subsequent non-blank
// lines before the next numbered line are folded into the same clause,
// which lets an author wrap a long clause across multiple source lines.
func parseLawLines(lines []string, startLineNo int) []RawLaw {
	var laws []RawLaw
	var current *RawLaw
	for i, line := range lines {
		lineNo := startLineNo + i + 1 // 1-based
		if m := lawLineRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				current.Text = strings.TrimSpace(current.Text)
				current.LineEnd = lineNo - 1
			}
			laws = append(laws, RawLaw{Text: m[2], LineStart: lineNo})
			current = &laws[len(laws)-1]
			continue
		}
		if current != nil && strings.TrimSpace(line) != "" {
			current.Text += " " + strings.TrimSpace(line)
		}
	}
	if current != nil {
		current.Text = strings.TrimSpace(current.Text)
		current.LineEnd = startLineNo + len(lines)
	}
	return laws
}
