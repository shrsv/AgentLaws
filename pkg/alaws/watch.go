package alaws

import "github.com/athreyac4/agentlaws/internal/watcher"

// WatchEvent describes a single recompilation triggered by a source
// change (or the initial compile when watching starts).
type WatchEvent struct {
	ClusterPath string
	Book        *Book // nil only if the lawbook couldn't be read at all
	Err         error // non-nil if compilation failed; Book.Diagnostics() may still be non-empty
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
			out <- WatchEvent{
				ClusterPath: ev.ClusterPath,
				Book:        &Book{lawbook: ev.Result.Lawbook, diagnostics: diagnosticsFrom(ev.Result.Diagnostics)},
				Err:         ev.Err,
			}
		}
	}()

	return out, stop, nil
}
