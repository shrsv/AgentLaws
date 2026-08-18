// Package watcher implements the filesystem watch loop behind `alaws watch`
// (PLAN1 §27, §54): debounce -> recompile -> notify caller.
package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/shrsv/AgentLaws/internal/compiler"
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

// Event describes a single recompilation triggered by a source change (or
// the initial compile when watching starts).
type Event struct {
	ClusterPath string
	Result      compiler.Result
	Err         error // non-nil if compilation failed; Result may still hold partial diagnostics
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
				select {
				case events <- Event{ClusterPath: path, Result: result, Err: err}:
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
			return err
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
