// Package promptexpand expands {{ref:x}} directives in prompt templates,
// stitching in law/section/prompt text at compile time. It produces
// fully-resolved PromptTemplate objects with deterministic, hashable
// Template fields ready for rendering. See docs/prompt-book.md §4.
package promptexpand

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/shrsv/AgentLaws/internal/model"
	"github.com/shrsv/AgentLaws/internal/parser"
	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/internal/template"
	"github.com/shrsv/AgentLaws/internal/validator"
)

// refDirectiveRe matches {{ref:<token>}} inside prompt template bodies.
var refDirectiveRe = regexp.MustCompile(`\{\{ref:([^}]+)\}\}`)

// Expand takes a compiled Lawbook (with numbering already assigned) and
// the raw parsed prompts, and returns fully-expanded PromptTemplates plus
// any diagnostics produced during expansion. It also computes the
// PromptBacklinks reverse index.
func Expand(book model.Lawbook, raw []parser.ParsedPrompt) ([]model.PromptTemplate, map[string][]string, []validator.Diagnostic) {
	exp := &expander{
		book:    book,
		rawByID: make(map[string]*parser.ParsedPrompt, len(raw)),
		memo:    make(map[string]*expandResult, len(raw)),
	}
	for i := range raw {
		exp.rawByID[raw[i].ID] = &raw[i]
	}

	var diags []validator.Diagnostic
	prompts := make([]model.PromptTemplate, 0, len(raw))
	for i := range raw {
		pt, d := exp.expandOne(raw[i].ID, nil)
		diags = append(diags, d...)
		if pt != nil {
			prompts = append(prompts, *pt)
		}
	}

	// Build backlinks: anchor -> sorted prompt IDs
	backlinks := map[string][]string{}
	for _, p := range prompts {
		for _, anchor := range p.ReferencedAnchors {
			backlinks[anchor] = append(backlinks[anchor], p.ID)
		}
	}
	for k := range backlinks {
		sort.Strings(backlinks[k])
	}

	return prompts, backlinks, diags
}

// expander holds state for a single Expand call.
type expander struct {
	book    model.Lawbook
	rawByID map[string]*parser.ParsedPrompt
	memo    map[string]*expandResult // prompt ID -> result (memoized)
}

type expandResult struct {
	prompt *model.PromptTemplate
	diags  []validator.Diagnostic
}

// expandOne expands a single prompt by ID, with cycle detection and memoization.
// visiting is the set of prompt IDs currently on the DFS stack (for cycle detection).
func (e *expander) expandOne(id string, visiting map[string]bool) (*model.PromptTemplate, []validator.Diagnostic) {
	raw, ok := e.rawByID[id]
	if !ok {
		return nil, nil
	}

	// Cycle detection — must come before memo check so a prompt currently
	// being expanded (in the visiting set) is caught as a cycle even though
	// it has a sentinel entry in the memo.
	if visiting == nil {
		visiting = map[string]bool{}
	}
	if visiting[id] {
		return nil, []validator.Diagnostic{{
			Severity: validator.SeverityError,
			Code:     "circular-prompt-reference",
			Message:  fmt.Sprintf("%s: circular prompt reference detected", id),
			Source:   &raw.Source,
		}}
	}

	// Check memo for already-completed expansions
	if r, ok := e.memo[id]; ok {
		return r.prompt, r.diags
	}

	visiting[id] = true
	defer delete(visiting, id)

	// Set a sentinel to detect cycles during expansion
	e.memo[id] = &expandResult{} // temporary sentinel

	var diags []validator.Diagnostic
	var segments []model.PromptSegment
	var anchors []string

	// Scan for {{ref:x}} directives
	lastEnd := 0
	rawTemplate := raw.RawTemplate
	for _, m := range refDirectiveRe.FindAllStringSubmatchIndex(rawTemplate, -1) {
		// Add text before this match
		if m[0] > lastEnd {
			segments = append(segments, model.PromptSegment{
				Kind: model.SegmentText,
				Text: rawTemplate[lastEnd:m[0]],
			})
		}

		token := rawTemplate[m[2]:m[3]]
		seg, anchor, subDiags := e.resolveRef(token, raw.Source, visiting)
		diags = append(diags, subDiags...)
		segments = append(segments, seg)
		if anchor != "" {
			anchors = append(anchors, anchor)
		}
		// Also collect transitive anchors from prompt refs
		if seg.Kind == model.SegmentPromptRef {
			if r, ok := e.memo[seg.RefAnchor]; ok && r.prompt != nil {
				anchors = append(anchors, r.prompt.ReferencedAnchors...)
			}
		}

		lastEnd = m[1]
	}

	// Add remaining text
	if lastEnd < len(rawTemplate) {
		segments = append(segments, model.PromptSegment{
			Kind: model.SegmentText,
			Text: rawTemplate[lastEnd:],
		})
	}

	// Build the flattened template
	var templateBuilder strings.Builder
	for _, seg := range segments {
		switch seg.Kind {
		case model.SegmentText:
			templateBuilder.WriteString(seg.Text)
		default:
			templateBuilder.WriteString(seg.Expanded)
		}
	}
	templateText := templateBuilder.String()

	// Deduplicate and sort anchors
	anchors = dedupSorted(anchors)

	// Extract vars
	vars := template.Vars(templateText)

	pt := &model.PromptTemplate{
		ID:                raw.ID,
		Title:             raw.Title,
		Commentary:        raw.Commentary,
		Segments:          segments,
		Template:          templateText,
		Vars:              vars,
		ReferencedAnchors: anchors,
		Source:            raw.Source,
	}

	e.memo[id] = &expandResult{prompt: pt, diags: diags}
	return pt, diags
}

// resolveRef resolves a single {{ref:token}} and returns the expanded segment.
func (e *expander) resolveRef(token string, src model.SourceRef, visiting map[string]bool) (model.PromptSegment, string, []validator.Diagnostic) {
	// First try resolver for laws/sections
	r, err := resolver.Resolve(e.book, token)
	if err == nil {
		switch r.Kind {
		case resolver.KindLaw:
			law := r.Law
			expanded := law.Number + " " + law.Text
			anchor := resolver.AnchorFor(r)
			label := law.Number
			return model.PromptSegment{
				Kind:      model.SegmentLawRef,
				RefToken:  token,
				RefAnchor: anchor,
				RefLabel:  label,
				Expanded:  expanded,
			}, anchor, nil

		case resolver.KindSection:
			sec := r.Section
			var lines []string
			for _, law := range sec.Laws {
				lines = append(lines, law.Number+" "+law.Text)
			}
			expanded := strings.Join(lines, "\n\n")
			anchor := resolver.AnchorFor(r)
			label := sec.Number + " " + sec.Title
			return model.PromptSegment{
				Kind:      model.SegmentSectionRef,
				RefToken:  token,
				RefAnchor: anchor,
				RefLabel:  label,
				Expanded:  expanded,
			}, anchor, nil
		}
	}

	// Try prompt-to-prompt resolution
	if raw, ok := e.rawByID[token]; ok {
		pt, subDiags := e.expandOne(token, visiting)
		if pt != nil {
			return model.PromptSegment{
				Kind:      model.SegmentPromptRef,
				RefToken:  token,
				RefAnchor: token,
				RefLabel:  raw.Title,
				Expanded:  pt.Template,
			}, token, subDiags
		}
		// If expansion failed (cycle), return the diags
		return model.PromptSegment{
			Kind:     model.SegmentText,
			Text:     fmt.Sprintf("{{ref:%s}}", token),
		}, "", subDiags
	}

	// Unresolved — leave as literal text and report diagnostic
	return model.PromptSegment{
		Kind:     model.SegmentText,
		Text:     fmt.Sprintf("{{ref:%s}}", token),
	}, "", []validator.Diagnostic{{
		Severity: validator.SeverityError,
		Code:     "dangling-prompt-reference",
		Message:  fmt.Sprintf("{{ref:%s}} does not resolve to any law, section, or prompt", token),
		Source:   &src,
	}}
}

// dedupSorted returns s sorted and deduplicated.
func dedupSorted(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	sort.Strings(s)
	j := 0
	for i := 1; i < len(s); i++ {
		if s[i] != s[j] {
			j++
			s[j] = s[i]
		}
	}
	return s[:j+1]
}
