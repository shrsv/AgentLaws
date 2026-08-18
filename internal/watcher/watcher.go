// Package watcher implements the filesystem watch loop behind `alaws watch`
// (PLAN1 §27, §54): debounce -> validate -> compile -> notify UI.
package watcher

import "errors"

// ErrNotImplemented is returned by every stub in this package until the
// watcher is implemented per PLAN1 §64 Milestone 8.
var ErrNotImplemented = errors.New("watcher: not implemented")

// Event describes a single recompilation triggered by a source change.
type Event struct {
	ClusterPath string
	Err         error // non-nil if compilation/validation failed
}

// Watch monitors alaws.toml and *.md/*.mdx files under path and sends an
// Event on the returned channel after each debounced recompilation. The
// returned stop function stops watching.
func Watch(path string) (events <-chan Event, stop func(), err error) {
	return nil, nil, ErrNotImplemented
}
