package alaws

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
)

// SearchMatch locates a single hit inside a compiled Lawbook.
type SearchMatch struct {
	SectionID    string `json:"sectionId"`
	SectionTitle string `json:"sectionTitle"`
	SectionNum   string `json:"sectionNumber"`
	SourcePath   string `json:"sourcePath"`
	LawIndex     int    `json:"lawIndex"`  // -1 for commentary/title matches
	LawNumber    string `json:"lawNumber"` // "" for commentary/title matches
	Line         int    `json:"line"`      // 1-based line in source file
	Before       string `json:"before"`    // context line before the match
	Match        string `json:"match"`     // the matching line
	After        string `json:"after"`     // context line after the match
}

// SearchOpts configures a search over a compiled Lawbook.
type SearchOpts struct {
	CaseSensitive bool
	WholeWord     bool
	Regex         bool
	SectionIDs    []string // empty → search all sections
}

// Search compiles the book at path and returns every line that matches q
// according to opts. The search covers section titles, commentary, and law
// text. It returns results even when the book has compilation errors, so
// partial/broken books can still be searched.
func Search(path, q string, opts SearchOpts) ([]SearchMatch, error) {
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}

	b, compileErr := Compile(path)
	if b == nil {
		return nil, compileErr
	}
	lb := b.Lawbook()

	var matcher func(string) []int
	if opts.Regex {
		flags := ""
		if !opts.CaseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + q)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		matcher = func(s string) []int {
			idxs := re.FindAllStringIndex(s, -1)
			out := make([]int, len(idxs))
			for i, loc := range idxs {
				out[i] = loc[0]
			}
			return out
		}
	} else {
		pattern := q
		text := q
		if !opts.CaseSensitive {
			pattern = strings.ToLower(q)
		}
		matcher = func(s string) []int {
			haystack := s
			if !opts.CaseSensitive {
				haystack = strings.ToLower(s)
			}
			var hits []int
			start := 0
			for {
				idx := strings.Index(haystack[start:], pattern)
				if idx < 0 {
					break
				}
				abs := start + idx
				if opts.WholeWord {
					if abs > 0 && isWordChar(s[abs-1]) {
						start = abs + 1
						continue
					}
					end := abs + len(text)
					if end < len(s) && isWordChar(s[end]) {
						start = abs + 1
						continue
					}
				}
				hits = append(hits, abs)
				start = abs + 1
			}
			return hits
		}
	}

	scopeSet := map[string]bool{}
	if len(opts.SectionIDs) > 0 {
		for _, id := range opts.SectionIDs {
			scopeSet[id] = true
		}
	}

	var results []SearchMatch
	for _, sec := range lb.Sections {
		if len(scopeSet) > 0 && !scopeSet[sec.ID] {
			continue
		}
		results = searchSection(sec, matcher, results)
	}
	return results, nil
}

func searchSection(sec model.Section, matcher func(string) []int, results []SearchMatch) []SearchMatch {
	lines := strings.Split(sec.Commentary, "\n")
	baseLine := sec.Source.LineStart

	titleHits := matcher(sec.Title)
	if len(titleHits) > 0 {
		afterLine := ""
		if len(lines) > 0 {
			afterLine = lines[0]
		}
		for range titleHits {
			results = append(results, SearchMatch{
				SectionID:    sec.ID,
				SectionTitle: sec.Title,
				SectionNum:   sec.Number,
				SourcePath:   sec.Source.Path,
				LawIndex:     -1,
				Line:         baseLine,
				Before:       "",
				Match:        sec.Title,
				After:        afterLine,
			})
		}
	}

	for i, line := range lines {
		if len(matcher(line)) == 0 {
			continue
		}
		before := ""
		if i > 0 {
			before = lines[i-1]
		}
		after := ""
		if i < len(lines)-1 {
			after = lines[i+1]
		}
		for range matcher(line) {
			results = append(results, SearchMatch{
				SectionID:    sec.ID,
				SectionTitle: sec.Title,
				SectionNum:   sec.Number,
				SourcePath:   sec.Source.Path,
				LawIndex:     -1,
				Line:         baseLine + i + 1,
				Before:       before,
				Match:        line,
				After:        after,
			})
		}
	}

	for _, law := range sec.Laws {
		lawLines := strings.Split(law.Text, "\n")
		for i, line := range lawLines {
			if len(matcher(line)) == 0 {
				continue
			}
			before := ""
			if i > 0 {
				before = lawLines[i-1]
			}
			after := ""
			if i < len(lawLines)-1 {
				after = lawLines[i+1]
			}
			for range matcher(line) {
				results = append(results, SearchMatch{
					SectionID:    sec.ID,
					SectionTitle: sec.Title,
					SectionNum:   sec.Number,
					SourcePath:   sec.Source.Path,
					LawIndex:     law.Index,
					LawNumber:    law.Number,
					Line:         law.Source.LineStart + i,
					Before:       before,
					Match:        line,
					After:        after,
				})
			}
		}
	}
	return results
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
