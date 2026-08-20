package alaws

import "github.com/shrsv/AgentLaws/internal/watcher"

// WatchEvent describes a single recompilation triggered by a source
// change (or the initial compile when watching starts).
//
// RenderedSections and RenderedPrompts contain pre-rendered HTML fragments
// for every section and prompt template, keyed by ID. These are computed
// by the watcher so callers (e.g. a live web UI) can update without a
// second round-trip. The __BOOK_PATH__ placeholder in alaws: link hrefs
// is already replaced with the URL-encoded book path.
type WatchEvent struct {
	ClusterPath     string
	Book            *Book // nil only if the lawbook couldn't be read at all
	Err             error // non-nil if compilation failed; Book.Diagnostics() may still be non-empty
	RenderedSections map[string]RenderedSection
	RenderedPrompts  map[string]RenderedPrompt
}

// Watch monitors alaws.toml and *.md/*.mdx files under path (including
// files in directories created after Watch starts) and sends a WatchEvent
// on the returned channel after each debounced recompilation, plus one
// immediately for the initial compile. The returned stop function stops
// watching and closes the channel. This is the library entry point behind
// `alaws watch` (docs/PLAN1.md §27, §54) - a Go program can watch a book
// the same way the CLI does.
func Watch(path string) (<-chan WatchEvent, func(), error) {
	events, stop, err := watcher.Watch(path)
	if err != nil {
		return nil, nil, err
	}

	out := make(chan WatchEvent)
	go func() {
		defer close(out)
		for ev := range events {
			we := WatchEvent{
				ClusterPath: ev.ClusterPath,
				Book:        &Book{lawbook: ev.Result.Lawbook, diagnostics: diagnosticsFrom(ev.Result.Diagnostics)},
				Err:         ev.Err,
			}
			// Pass through pre-rendered HTML fragments from the watcher.
			if ev.RenderedSections != nil {
				we.RenderedSections = make(map[string]RenderedSection, len(ev.RenderedSections))
				for id, rs := range ev.RenderedSections {
					we.RenderedSections[id] = RenderedSection{
						CommentaryHTML: rs.CommentaryHTML,
						LawHTML:        rs.LawHTML,
					}
				}
			}
			if ev.RenderedPrompts != nil {
				we.RenderedPrompts = make(map[string]RenderedPrompt, len(ev.RenderedPrompts))
				for id, rp := range ev.RenderedPrompts {
					we.RenderedPrompts[id] = RenderedPrompt{
						CommentaryHTML: rp.CommentaryHTML,
						TemplateHTML:   rp.TemplateHTML,
						CompactHTML:    rp.CompactHTML,
					}
				}
			}
			out <- we
		}
	}()

	return out, stop, nil
}
