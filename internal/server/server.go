// Package server serves the embedded Preact UI and its local API. See
// docs/PLAN1.md §28, §53.
//
// The Lawbook/diagnostics/ordering-update API endpoints are not implemented
// yet - they depend on the compiler and ordering packages (PLAN1 §64
// Milestones 2-4, 9). This package currently only serves the static UI
// shell, which is enough for `alaws serve`/`alaws watch` to be runnable end
// to end while those depend on API packages are filled in.
package server

import (
	"io/fs"
	"net/http"

	"github.com/athreyac4/agentlaws/web"
)

// Handler returns an http.Handler serving the embedded web/dist assets.
func Handler() (http.Handler, error) {
	assets, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(assets)), nil
}

// ListenAndServe starts the local UI/API server on addr (e.g. ":8420").
func ListenAndServe(addr string) error {
	handler, err := Handler()
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, handler)
}
