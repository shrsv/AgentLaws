// Package watcher implements the filesystem watch loop behind `alaws watch`
// (PLAN1 §27, §54): debounce -> recompile -> notify caller.
package watcher

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/shrsv/AgentLaws/internal/compiler"
	"github.com/shrsv/AgentLaws/internal/resolver"
	renderhtml "github.com/shrsv/AgentLaws/internal/renderer/html"
)

// debounceWindow avoids recompiling once per fsnotify event when an editor
// (or another `alaws` command) touches several files in quick succession -
// see PLAN1 §54.
const debounceWindow = 300 * time.Millisecond

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
	".alaws":       true, // avoid recompiling in response to our own build output
}

// RenderedSection holds pre-rendered HTML fragments for a single section,
// keyed by law citation number. Used by the SSE watch stream so the web UI
// can update without a second round-trip.
type RenderedSection struct {
	CommentaryHTML string            `json:"CommentaryHTML"`
	LawHTML        map[string]string `json:"LawHTML"`
}

// RenderedPrompt holds pre-rendered HTML fragments for a single prompt
// template. TemplateHTML is the expanded version; CompactHTML shows ref
// directives as alaws: links.
type RenderedPrompt struct {
	CommentaryHTML string `json:"CommentaryHTML"`
	TemplateHTML   string `json:"TemplateHTML"`
	CompactHTML    string `json:"CompactHTML"`
}

// Event describes a single recompilation triggered by a source change (or
// the initial compile when watching starts).
type Event struct {
	ClusterPath     string
	Result          compiler.Result
	Err             error // non-nil if compilation failed; Result may still hold partial diagnostics
	RenderedSections map[string]RenderedSection
	RenderedPrompts  map[string]RenderedPrompt
}

// Watch monitors alaws.toml and *.md/*.mdx files under path (including
// files in directories created after Watch starts) and sends an Event on
// the returned channel after each debounced recompilation, plus one
// immediately for the initial compile. The returned stop function stops
// watching and closes the channel.
func Watch(path string) (<-chan Event, func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	if err := addDirsRecursive(w, path); err != nil {
		w.Close()
		return nil, nil, err
	}

	events := make(chan Event)
	stopCh := make(chan struct{})

	go func() {
		defer w.Close()
		defer close(events)

		// A debounced fsnotify event (or the initial compile) only ever
		// requests a recompile through this buffered channel; the actual
		// compiler.Compile call and the send on `events` both happen
		// below, in this single goroutine. That keeps every send on
		// `events` and every close(events) in the same goroutine, so a
		// pending recompile can never race a send against stop() closing
		// the channel out from under it.
		trigger := make(chan struct{}, 1)
		request := func() {
			select {
			case trigger <- struct{}{}:
			default:
			}
		}

		var debounce *time.Timer
		defer func() {
			if debounce != nil {
				debounce.Stop()
			}
		}()

		request() // initial compile

		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if !relevant(ev.Name) {
					continue
				}
				if ev.Op&fsnotify.Create != 0 {
					if info, statErr := os.Stat(ev.Name); statErr == nil && info.IsDir() {
						_ = addDirsRecursive(w, ev.Name)
					}
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(debounceWindow, request)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			case <-trigger:
				result, err := compiler.Compile(path, compiler.Options{})

				// Compute rendered HTML fragments so the web UI can update
				// without a second round-trip to /api/book/compile.
				ev := Event{ClusterPath: path, Result: result, Err: err}
				ev.RenderedSections, ev.RenderedPrompts = computeRendered(path, result)

				select {
				case events <- ev:
				case <-stopCh:
					return
				}
			case <-stopCh:
				return
			}
		}
	}()

	stop := func() { close(stopCh) }
	return events, stop, nil
}

func relevant(name string) bool {
	if filepath.Base(name) == "alaws.toml" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".mdx"
}

func addDirsRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			return filepath.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if p != root && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		return w.Add(p)
	})
}

// computeRendered builds pre-rendered HTML fragments for sections and
// prompt templates from a compilation result. The __BOOK_PATH__ placeholder
// is replaced here so the SSE payload contains navigable hash routes.
func computeRendered(path string, result compiler.Result) (map[string]RenderedSection, map[string]RenderedPrompt) {
	lb := result.Lawbook
	escapedBookPath := url.PathEscape(path)

	resolve := func(token string) (string, bool) {
		r, err := resolver.Resolve(lb, token)
		if err != nil {
			return "", false
		}
		switch r.Kind {
		case resolver.KindLaw:
			lawAnchor := r.Law.Slug
			if lawAnchor == "" {
				lawAnchor = fmt.Sprintf("%d", r.Law.Index)
			}
			return fmt.Sprintf("#/books/__BOOK_PATH__/%s~%s", r.Law.SectionID, lawAnchor), true
		case resolver.KindSection:
			return fmt.Sprintf("#/books/__BOOK_PATH__/%s", r.Section.ID), true
		case resolver.KindPrompt:
			return fmt.Sprintf("#/books/__BOOK_PATH__/prompts/%s", r.Prompt.ID), true
		}
		return "", false
	}

	replaceBookPath := func(s string) string {
		return strings.ReplaceAll(s, "__BOOK_PATH__", escapedBookPath)
	}

	sections := make(map[string]RenderedSection, len(lb.Sections))
	for _, s := range lb.Sections {
		commentaryHTML, err := renderhtml.RenderFragment(s.Commentary, resolve)
		if err != nil {
			continue
		}
		lawHTML := make(map[string]string, len(s.Laws))
		for _, law := range s.Laws {
			frag, err := renderhtml.RenderFragment(law.Text, resolve)
			if err != nil {
				continue
			}
			lawHTML[law.Number] = replaceBookPath(frag)
		}
		sections[s.ID] = RenderedSection{
			CommentaryHTML: replaceBookPath(commentaryHTML),
			LawHTML:        lawHTML,
		}
	}

	prompts := make(map[string]RenderedPrompt, len(lb.Prompts))
	for _, p := range lb.Prompts {
		commentaryHTML, err := renderhtml.RenderFragment(p.Commentary, resolve)
		if err != nil {
			continue
		}
		templateHTML, err := renderhtml.RenderFragment(p.Template, resolve)
		if err != nil {
			continue
		}
		compactMD := resolver.PromptDisplayText(p, false)
		compactHTML, err := renderhtml.RenderFragment(compactMD, resolve)
		if err != nil {
			continue
		}
		prompts[p.ID] = RenderedPrompt{
			CommentaryHTML: replaceBookPath(commentaryHTML),
			TemplateHTML:   replaceBookPath(templateHTML),
			CompactHTML:    replaceBookPath(compactHTML),
		}
	}

	return sections, prompts
}
