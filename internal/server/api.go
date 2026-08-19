// Package server's API is a thin JSON+SSE wrapper over pkg/alaws - the
// same architectural rule the CLI follows (docs/PLAN1.md §52): every
// handler here does request parsing, one pkg/alaws call, and response
// encoding, and nothing else. This is what gives the web UI parity with
// the CLI and the Go library by construction rather than by convention.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shrsv/AgentLaws/internal/resolver"
	"github.com/shrsv/AgentLaws/pkg/alaws"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeError maps an error to a status code and a structured JSON body.
// Not-found citations/IDs (internal/resolver.ErrNotFound, which pkg/alaws
// surfaces directly) become 404; everything else from pkg/alaws is a
// client input problem from the API's point of view (a bad path, a
// missing parent, malformed source) and becomes 400.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "bad-request"
	if errors.Is(err, resolver.ErrNotFound) {
		status = http.StatusNotFound
		code = "not-found"
	}
	writeJSON(w, status, map[string]string{"error": err.Error(), "code": code})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed", "code": "bad-request"})
}

// registerAPI mounts every /api/ handler on mux.
func registerAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/books", handleBooks)
	mux.HandleFunc("/api/book", handleBook)
	mux.HandleFunc("/api/book/compile", handleCompile)
	mux.HandleFunc("/api/book/export", handleExport)
	mux.HandleFunc("/api/book/render", handleRender)
	mux.HandleFunc("/api/book/chapters", handleChapters)
	mux.HandleFunc("/api/book/sections", handleSections)
	mux.HandleFunc("/api/book/move", handleMove)
	mux.HandleFunc("/api/book/laws", handleLaws)
	mux.HandleFunc("/api/book/watch", handleWatch)
	mux.HandleFunc("/api/book/manifest", handleManifest)
	mux.HandleFunc("/api/book/history", handleHistory)
	mux.HandleFunc("/api/book/log", handleLog)
	mux.HandleFunc("/api/book/verify", handleVerify)
	mux.HandleFunc("/api/book/commit-detail", handleCommitDetail)
	mux.HandleFunc("/api/book/source", handleSource)
	mux.HandleFunc("/api/book/search", handleSearch)
	mux.HandleFunc("/api/meta/operations", handleOperations)
	mux.HandleFunc("/api/meta/root", handleRoot)
	mux.HandleFunc("/api/export", handleExportAll)
}

// GET /api/meta/root - the discovery root the CLI was started with
// (docs/PLAN1.md §32), so the UI's home view can show what it's looking
// at instead of an opaque ".".
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"root": absRoot()})
}

// GET /api/export?root=&format=html|pdf&title=
func handleExportAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	exportRoot := q.Get("root")
	if exportRoot == "" {
		exportRoot = root
	}
	format := q.Get("format")
	if format == "" {
		format = "html"
	}
	title := q.Get("title")
	if title == "" {
		title = "Combined Lawbook"
	}

	books, err := alaws.CompileAll(exportRoot)
	if len(books) == 0 {
		writeError(w, err)
		return
	}

	switch format {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = alaws.RenderCombinedHTML(w, title, books)
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		_ = alaws.RenderCombinedPDF(w, title, books)
	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_ = alaws.RenderCombinedMarkdown(w, title, books)
	default:
		writeError(w, fmt.Errorf("unknown format %q, want html, pdf, or md", format))
	}
}

// GET  /api/books?root=. - discover books
// POST /api/books {path, title} - create a book
func handleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		discoverRoot := r.URL.Query().Get("root")
		if discoverRoot == "" {
			discoverRoot = root
		}
		books, err := alaws.Discover(discoverRoot)
		if err != nil {
			writeError(w, err)
			return
		}
		if books == nil {
			books = []alaws.BookInfo{}
		}
		writeJSON(w, http.StatusOK, books)

	case http.MethodPost:
		var req struct {
			Path  string `json:"path"`
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, err)
			return
		}
		if err := alaws.CreateBook(req.Path, req.Title); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"path": req.Path})

	default:
		methodNotAllowed(w)
	}
}

// GET /api/book?path=
func handleBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	book := r.URL.Query().Get("path")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	title, err := alaws.Title(book)
	if err != nil {
		writeError(w, err)
		return
	}
	tree, err := alaws.Tree(book)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"title": title, "sections": tree})
}

// GET /api/book/compile?path=
func handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	book := r.URL.Query().Get("path")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	b, err := alaws.Compile(book)
	// rendered lets the UI show commentary/law text as formatted HTML
	// (code spans, lists, highlighted code blocks) instead of raw
	// Markdown - the same pipeline the HTML export uses, so the two stay
	// visually identical (docs/PLAN1.md §28).
	rendered, renderErr := b.RenderedSections()
	if renderErr != nil && err == nil {
		err = renderErr
	}
	// Diagnostics matter even when err != nil (docs/PLAN1.md §20), so this
	// endpoint always returns 200 with both; the caller checks "ok".
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          err == nil,
		"error":       errString(err),
		"lawbook":     b.Lawbook(),
		"diagnostics": b.Diagnostics(),
		"rendered":    rendered,
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// GET /api/book/export?path=&format=html|pdf
func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	book := r.URL.Query().Get("path")
	format := r.URL.Query().Get("format")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	b, err := alaws.Compile(book)
	if err != nil {
		writeError(w, err)
		return
	}
	switch format {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = b.RenderHTML(w)
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		_ = b.RenderPDF(w)
	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_ = b.RenderMarkdown(w)
	default:
		writeError(w, fmt.Errorf("unknown format %q, want html, pdf, or md", format))
	}
}

// GET /api/book/render?path=&section=&law=&all=1&var=key:value&onMissing=error|keep|empty
func handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	book := q.Get("path")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	b, err := alaws.Load(book)
	if err != nil {
		writeError(w, err)
		return
	}

	sel := alaws.Selector{All: q.Get("all") == "1" || q.Get("all") == "true"}
	if s := q.Get("section"); s != "" {
		sel.SectionIDs = []string{s}
	}
	if l := q.Get("law"); l != "" {
		sel.Citations = []string{l}
	}
	laws, err := b.Laws(sel)
	if err != nil {
		writeError(w, err)
		return
	}

	vars := map[string]string{}
	for _, kv := range q["var"] {
		k, v, ok := strings.Cut(kv, ":")
		if !ok {
			writeError(w, fmt.Errorf("--var must be in key:value form, got %q", kv))
			return
		}
		vars[k] = v
	}

	policy := alaws.MissingError
	switch q.Get("onMissing") {
	case "keep":
		policy = alaws.MissingKeepPlaceholder
	case "empty":
		policy = alaws.MissingEmpty
	}

	text, err := laws.Render(alaws.RenderOptions{Vars: vars, OnMissing: policy})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

type createSectionRequest struct {
	Book     string `json:"book"`
	File     string `json:"file"`
	Title    string `json:"title"`
	ID       string `json:"id"`
	Parent   string `json:"parent"`
	Level    int    `json:"level"`
	After    string `json:"after"`
	Before   string `json:"before"`
	Position int    `json:"position"`
}

func (r createSectionRequest) placement() alaws.Placement {
	return alaws.Placement{After: r.After, Before: r.Before, Position: r.Position}
}

// POST /api/book/chapters {book, file, title, id, after, before, position}
func handleChapters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req createSectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	if err := alaws.CreateChapter(req.Book, req.File, req.Title, req.ID, req.placement()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": req.ID})
}

// POST /api/book/sections {book, file, title, id, parent, level, after, before, position}
func handleSections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createSectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, err)
			return
		}
		if err := alaws.CreateSection(req.Book, req.File, req.Title, req.ID, req.Parent, req.Level, req.placement()); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": req.ID})

	case http.MethodDelete:
		q := r.URL.Query()
		book, id := q.Get("book"), q.Get("id")
		force := q.Get("force") == "1" || q.Get("force") == "true"
		kind := q.Get("kind") // "chapter" or "section"
		var err error
		if kind == "chapter" {
			err = alaws.RemoveChapter(book, id, force)
		} else {
			err = alaws.RemoveSection(book, id, force)
		}
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"removed": id})

	default:
		methodNotAllowed(w)
	}
}

type moveRequest struct {
	Book        string `json:"book"`
	Kind        string `json:"kind"` // "chapter" or "section"
	ID          string `json:"id"`
	NewParentID string `json:"newParentId"` // sections only
	After       string `json:"after"`
	Before      string `json:"before"`
	Position    int    `json:"position"`
}

// POST /api/book/move {book, kind, id, newParentId, after, before, position}
// This is the drag-and-drop reorder endpoint (docs/PLAN1.md §29).
func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	p := alaws.Placement{After: req.After, Before: req.Before, Position: req.Position}
	var err error
	if req.Kind == "chapter" {
		err = alaws.MoveChapter(req.Book, req.ID, p)
	} else {
		err = alaws.MoveSection(req.Book, req.ID, req.NewParentID, p)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"moved": req.ID})
}

type addLawRequest struct {
	Book      string `json:"book"`
	SectionID string `json:"sectionId"`
	Text      string `json:"text"`
	After     int    `json:"after"`
}

// GET /api/book/laws?book=&sectionId=
// POST /api/book/laws {book, sectionId, text, after}
// DELETE /api/book/laws?book=&sectionId=&number=&force=
func handleLaws(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		laws, err := alaws.ListLaws(q.Get("book"), q.Get("sectionId"))
		if err != nil {
			writeError(w, err)
			return
		}
		if laws == nil {
			laws = []string{}
		}
		writeJSON(w, http.StatusOK, laws)

	case http.MethodPost:
		var req addLawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, err)
			return
		}
		if err := alaws.AddLaw(req.Book, req.SectionID, req.Text, req.After); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"sectionId": req.SectionID})

	case http.MethodDelete:
		q := r.URL.Query()
		n, err := strconv.Atoi(q.Get("number"))
		if err != nil {
			writeError(w, fmt.Errorf("number must be an integer, got %q", q.Get("number")))
			return
		}
		force := q.Get("force") == "1" || q.Get("force") == "true"
		if err := alaws.RemoveLaw(q.Get("book"), q.Get("sectionId"), n, force); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"sectionId": q.Get("sectionId")})

	default:
		methodNotAllowed(w)
	}
}

// GET /api/book/manifest?path=
func handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	book := r.URL.Query().Get("path")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	b, err := alaws.Compile(book)
	if err != nil {
		writeError(w, err)
		return
	}
	m, err := b.Manifest()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// GET /api/book/history?path=&citation=
func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	book := q.Get("path")
	citation := q.Get("citation")
	if book == "" || citation == "" {
		writeError(w, errors.New("path and citation are required"))
		return
	}
	b, err := alaws.Compile(book)
	if err != nil {
		writeError(w, err)
		return
	}
	hist, err := b.History(citation)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hist)
}

// GET /api/book/log?path=&limit=
func handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	book := q.Get("path")
	if book == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	limit := 0
	if s := q.Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	changes, err := alaws.Log(book, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	if changes == nil {
		changes = []alaws.LogEntry{}
	}
	writeJSON(w, http.StatusOK, changes)
}

// POST /api/book/verify {path, manifest}
func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Path     string         `json:"path"`
		Manifest alaws.Manifest `json:"manifest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	b, err := alaws.Compile(req.Path)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := alaws.Verify(req.Manifest, b); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/book/commit-detail?path=&commit=
func handleCommitDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	book := q.Get("path")
	commit := q.Get("commit")
	if book == "" || commit == "" {
		writeError(w, errors.New("path and commit are required"))
		return
	}
	detail, err := alaws.CommitDetailFor(book, commit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// GET /api/book/source?path=&lineStart=&lineEnd=
func handleSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	srcPath := q.Get("path")
	if srcPath == "" {
		writeError(w, errors.New("path is required"))
		return
	}
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		writeError(w, err)
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, err)
		return
	}
	lines := strings.Split(string(data), "\n")
	start := 1
	end := len(lines)
	if s := q.Get("lineStart"); s != "" {
		if n, convErr := strconv.Atoi(s); convErr == nil && n > 0 {
			start = n
		}
	}
	if s := q.Get("lineEnd"); s != "" {
		if n, convErr := strconv.Atoi(s); convErr == nil && n > 0 {
			end = n
		}
	}
	if start > len(lines) {
		start = len(lines)
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	snippet := strings.Join(lines[start-1:end], "\n")
	writeJSON(w, http.StatusOK, map[string]any{
		"path":      srcPath,
		"lineStart": start,
		"lineEnd":   end,
		"content":   snippet,
	})
}

// GET /api/book/search?q=&path=&caseSensitive=&wholeWord=&regex=&sectionId=
func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	book := q.Get("path")
	query := q.Get("q")
	if book == "" || query == "" {
		writeError(w, errors.New("path and q are required"))
		return
	}
	opts := alaws.SearchOpts{
		CaseSensitive: q.Get("caseSensitive") == "1" || q.Get("caseSensitive") == "true",
		WholeWord:     q.Get("wholeWord") == "1" || q.Get("wholeWord") == "true",
		Regex:         q.Get("regex") == "1" || q.Get("regex") == "true",
		SectionIDs:    q["sectionId"],
	}
	matches, err := alaws.Search(book, query, opts)
	if err != nil {
		writeError(w, err)
		return
	}
	if matches == nil {
		matches = []alaws.SearchMatch{}
	}
	writeJSON(w, http.StatusOK, matches)
}
