// Package server serves the embedded Preact UI and its local JSON+SSE API
// (api.go, watch.go, operations.go). See docs/PLAN1.md §28, §53.
package server

import (
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/shrsv/AgentLaws/web"
)

// root is the discovery root the UI should default to and display, set by
// the CLI (`alaws serve`/`watch`/`ui` all pass their --root) before
// starting the server. It exists so the web UI's book picker searches and
// shows the same root the CLI was pointed at, instead of always assuming
// the server process's own working directory (docs/PLAN1.md §32).
var root = "."

// SetRoot sets the discovery root exposed at GET /api/meta/root and used
// as the default for GET /api/books when its root query param is empty.
func SetRoot(r string) {
	if r != "" {
		root = r
	}
}

// absRoot resolves root to an absolute path for display - "." tells a
// user nothing about what they're looking at, but the real path does.
func absRoot() string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return abs
}

// staticHandler returns an http.Handler serving the embedded web/dist
// assets.
func staticHandler() (http.Handler, error) {
	assets, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(assets)), nil
}

// Handler returns the full local server: every /api/ route (Part 3) plus
// the embedded UI as a fallback for everything else.
func Handler() (http.Handler, error) {
	static, err := staticHandler()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	registerAPI(mux)
	mux.Handle("/", static)
	return mux, nil
}

// ListenAndServe starts the local UI/API server on addr (e.g. ":8420").
func ListenAndServe(addr string) error {
	handler, err := Handler()
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, handler)
}
