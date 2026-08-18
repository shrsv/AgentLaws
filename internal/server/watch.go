package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

// GET /api/book/watch?path= (Server-Sent Events)
//
// Streams a JSON event on every recompile, reusing pkg/alaws.Watch - the
// same debounced file-watch loop `alaws watch` runs, so the web UI's
// watch panel behaves identically to the CLI's (docs/PLAN1.md §27, §54).
// No client library is needed: the browser's built-in EventSource speaks
// this format natively.
func handleWatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	book := r.URL.Query().Get("path")
	if book == "" {
		writeError(w, fmt.Errorf("path is required"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("streaming unsupported"))
		return
	}

	events, stop, err := alaws.Watch(book)
	if err != nil {
		writeError(w, err)
		return
	}
	defer stop()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	enc := json.NewEncoder(w)
	ctx := r.Context()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			fmt.Fprint(w, "data: ")
			_ = enc.Encode(map[string]any{
				"clusterPath": ev.ClusterPath,
				"ok":          ev.Err == nil,
				"error":       errString(ev.Err),
				"lawbook":     ev.Book.Lawbook(),
				"diagnostics": ev.Book.Diagnostics(),
			})
			fmt.Fprint(w, "\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
