package server

import "net/http"

// Param describes one input to an Operation. Kind is a UI hint for how the
// Playground should render the field ("book", "id", "citation", "text",
// "int", "bool", "select:error|keep|empty"), not a wire type.
type Param struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Operation documents one capability available identically through the
// CLI, the pkg/alaws Go library, and this HTTP API. CLITemplate and
// GoTemplate use {param} placeholders the Playground UI substitutes with
// whatever the user actually entered - this is the single source of truth
// behind "teach me the CLI/library while I use the UI" (docs/PLAN1.md
// §32): one Operation literal per capability, co-located with the handler
// it describes, rather than duplicated strings scattered through the
// frontend.
type Operation struct {
	ID          string  `json:"id"`
	Group       string  `json:"group"`
	Summary     string  `json:"summary"`
	Method      string  `json:"method"`
	Path        string  `json:"path"`
	Params      []Param `json:"params"`
	CLITemplate string  `json:"cliTemplate"`
	GoTemplate  string  `json:"goTemplate"`
}

// Operations is the full manifest served at GET /api/meta/operations.
var Operations = []Operation{
	{
		ID: "books.list", Group: "Books", Summary: "Discover every book under a root directory",
		Method: http.MethodGet, Path: "/api/books",
		Params:      []Param{{Name: "root", Kind: "text", Description: "directory to search (default .)"}},
		CLITemplate: "alaws books list --root {root}",
		GoTemplate:  `alaws.Discover({root})`,
	},
	{
		ID: "books.create", Group: "Books", Summary: "Create a new book",
		Method: http.MethodPost, Path: "/api/books",
		Params: []Param{
			{Name: "path", Kind: "text", Required: true, Description: "directory for the new book"},
			{Name: "title", Kind: "text", Description: "book title"},
		},
		CLITemplate: "alaws books create {path} --title {title}",
		GoTemplate:  `alaws.CreateBook({path}, {title})`,
	},
	{
		ID: "book.show", Group: "Books", Summary: "Show a book's title and chapter/section tree",
		Method: http.MethodGet, Path: "/api/book",
		Params:      []Param{{Name: "path", Kind: "book", Required: true, Description: "the book"}},
		CLITemplate: "alaws books show {path}",
		GoTemplate:  `alaws.Title({path}); alaws.Tree({path})`,
	},
	{
		ID: "book.compile", Group: "Compile", Summary: "Compile a book and report diagnostics",
		Method: http.MethodGet, Path: "/api/book/compile",
		Params:      []Param{{Name: "path", Kind: "book", Required: true, Description: "the book"}},
		CLITemplate: "alaws compile {path}",
		GoTemplate:  `book, err := alaws.Compile({path})`,
	},
	{
		ID: "book.render", Group: "Render", Summary: "Render selected laws, substituting {{variables}}",
		Method: http.MethodGet, Path: "/api/book/render",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "section", Kind: "id", Description: "render all laws in this section"},
			{Name: "law", Kind: "citation", Description: "render a single law by citation"},
			{Name: "all", Kind: "bool", Description: "render every law in the book"},
			{Name: "var", Kind: "vars", Description: "variables, key:value (repeatable)"},
			{Name: "onMissing", Kind: "select:error|keep|empty", Description: "missing-variable policy"},
		},
		CLITemplate: "alaws render --book {path} --section {section} --var {var} --on-missing {onMissing}",
		GoTemplate:  `book.Laws(alaws.Selector{...}).Render(alaws.RenderOptions{...})`,
	},
	{
		ID: "chapter.create", Group: "Chapters", Summary: "Create a new chapter (top-level section)",
		Method: http.MethodPost, Path: "/api/book/chapters",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "file", Kind: "text", Required: true, Description: "Markdown file path, relative to the book"},
			{Name: "title", Kind: "text", Required: true, Description: "chapter title"},
			{Name: "id", Kind: "id", Required: true, Description: "stable section ID"},
			{Name: "after", Kind: "id", Description: "insert after this ID"},
		},
		CLITemplate: "alaws chapter create {file} --book {book} --title {title} --id {id} --after {after}",
		GoTemplate:  `alaws.CreateChapter({book}, {file}, {title}, {id}, alaws.Placement{After: {after}})`,
	},
	{
		ID: "section.create", Group: "Sections", Summary: "Create a new section under a parent chapter",
		Method: http.MethodPost, Path: "/api/book/sections",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "file", Kind: "text", Required: true, Description: "Markdown file path, relative to the book"},
			{Name: "title", Kind: "text", Required: true, Description: "section title"},
			{Name: "id", Kind: "id", Required: true, Description: "stable section ID"},
			{Name: "parent", Kind: "id", Required: true, Description: "parent chapter ID"},
		},
		CLITemplate: "alaws section create {file} --book {book} --parent {parent} --title {title} --id {id}",
		GoTemplate:  `alaws.CreateSection({book}, {file}, {title}, {id}, {parent}, 0, alaws.Placement{})`,
	},
	{
		ID: "book.move", Group: "Reorder", Summary: "Move a chapter or section (drag-and-drop backend)",
		Method: http.MethodPost, Path: "/api/book/move",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "kind", Kind: "select:chapter|section", Required: true, Description: "what's being moved"},
			{Name: "id", Kind: "id", Required: true, Description: "the chapter/section to move"},
			{Name: "newParentId", Kind: "id", Description: "sections only: new parent chapter"},
			{Name: "before", Kind: "id", Description: "move before this ID"},
			{Name: "after", Kind: "id", Description: "move after this ID"},
		},
		CLITemplate: "alaws {kind} move {id} --book {book} --before {before} --after {after}",
		GoTemplate:  `alaws.MoveSection({book}, {id}, {newParentId}, alaws.Placement{Before: {before}, After: {after}})`,
	},
	{
		ID: "sections.remove", Group: "Reorder", Summary: "Remove a chapter or section from the ordering",
		Method: http.MethodDelete, Path: "/api/book/sections",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "kind", Kind: "select:chapter|section", Required: true, Description: "what's being removed"},
			{Name: "id", Kind: "id", Required: true, Description: "the chapter/section to remove"},
			{Name: "force", Kind: "bool", Description: "remove descendants too"},
		},
		CLITemplate: "alaws {kind} remove {id} --book {book} --force={force}",
		GoTemplate:  `alaws.RemoveSection({book}, {id}, {force})`,
	},
	{
		ID: "laws.add", Group: "Laws", Summary: "Append a numbered clause to a section",
		Method: http.MethodPost, Path: "/api/book/laws",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "sectionId", Kind: "id", Required: true, Description: "the section"},
			{Name: "text", Kind: "text", Required: true, Description: "the clause text"},
		},
		CLITemplate: "alaws law add {sectionId} {text} --book {book}",
		GoTemplate:  `alaws.AddLaw({book}, {sectionId}, {text}, 0)`,
	},
	{
		ID: "laws.list", Group: "Laws", Summary: "List a section's numbered clauses",
		Method: http.MethodGet, Path: "/api/book/laws",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "sectionId", Kind: "id", Required: true, Description: "the section"},
		},
		CLITemplate: "alaws law list {sectionId} --book {book}",
		GoTemplate:  `alaws.ListLaws({book}, {sectionId})`,
	},
	{
		ID: "laws.remove", Group: "Laws", Summary: "Remove a numbered clause from a section",
		Method: http.MethodDelete, Path: "/api/book/laws",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "sectionId", Kind: "id", Required: true, Description: "the section"},
			{Name: "number", Kind: "int", Required: true, Description: "1-based clause number"},
		},
		CLITemplate: "alaws law remove {sectionId} {number} --book {book}",
		GoTemplate:  `alaws.RemoveLaw({book}, {sectionId}, {number}, false)`,
	},
	{
		ID: "books.export", Group: "Books", Summary: "Export every book under a root as one combined document",
		Method: http.MethodGet, Path: "/api/export",
		Params: []Param{
			{Name: "root", Kind: "text", Description: "directory to search (default the server's root)"},
			{Name: "format", Kind: "select:html|pdf|md", Description: "combined document format"},
			{Name: "title", Kind: "text", Description: `document title (default "Combined Lawbook")`},
		},
		CLITemplate: "alaws export {root} --format {format} --title {title}",
		GoTemplate:  `books, _ := alaws.CompileAll({root}); alaws.RenderCombinedHTML(w, {title}, books)`,
	},
	{
		ID: "book.watch", Group: "Watch", Summary: "Stream live recompile events while editing",
		Method: http.MethodGet, Path: "/api/book/watch",
		Params:      []Param{{Name: "path", Kind: "book", Required: true, Description: "the book"}},
		CLITemplate: "alaws watch {path}",
		GoTemplate:  `events, stop, err := alaws.Watch({path})`,
	},
	{
		ID: "book.manifest", Group: "Provenance", Summary: "Get the provenance manifest for a compiled book",
		Method: http.MethodGet, Path: "/api/book/manifest",
		Params:      []Param{{Name: "path", Kind: "book", Required: true, Description: "the book"}},
		CLITemplate: "alaws sign {path} (then inspect .alaws/build/manifest.json)",
		GoTemplate:  `b, _ := alaws.Compile({path}); m, _ := b.Manifest()`,
	},
	{
		ID: "book.history", Group: "Provenance", Summary: "Get the Git history of a specific law",
		Method: http.MethodGet, Path: "/api/book/history",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "citation", Kind: "citation", Required: true, Description: "e.g. 2.5.3"},
		},
		CLITemplate: "alaws history {citation} --book {path}",
		GoTemplate:  `b, _ := alaws.Compile({path}); h, _ := b.History({citation})`,
	},
	{
		ID: "book.log", Group: "Provenance", Summary: "Get the chronological change log of a book",
		Method: http.MethodGet, Path: "/api/book/log",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "limit", Kind: "int", Description: "max commits (0 = all)"},
		},
		CLITemplate: "alaws log {path} --limit {limit}",
		GoTemplate:  `changes, _ := alaws.BookLog({path}, {limit})`,
	},
	{
		ID: "book.verify", Group: "Provenance", Summary: "Verify a book's compiled state against a signed manifest",
		Method: http.MethodPost, Path: "/api/book/verify",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "manifest", Kind: "json", Required: true, Description: "the manifest to verify against"},
		},
		CLITemplate: "alaws verify {path} --manifest manifest.json",
		GoTemplate:  `b, _ := alaws.Compile({path}); alaws.Verify(manifest, b)`,
	},
	{
		ID: "book.commit-detail", Group: "Provenance", Summary: "Get full detail of a commit (files + semantic diff)",
		Method: http.MethodGet, Path: "/api/book/commit-detail",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "commit", Kind: "text", Required: true, Description: "commit hash (full or short)"},
		},
		CLITemplate: "alaws log {path} (then inspect a specific commit)",
		GoTemplate:  `detail, _ := alaws.CommitDetailFor({path}, {commit})`,
	},
	{
		ID: "prompt.list", Group: "Prompts", Summary: "List prompt templates in a book",
		Method: http.MethodGet, Path: "/api/book/compile",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
		},
		CLITemplate: "alaws prompt list {path}",
		GoTemplate:  `book.Prompts()`,
	},
	{
		ID: "prompt.render", Group: "Prompts", Summary: "Render a prompt template with variable substitution",
		Method: http.MethodGet, Path: "/api/book/prompt/render",
		Params: []Param{
			{Name: "path", Kind: "book", Required: true, Description: "the book"},
			{Name: "id", Kind: "id", Required: true, Description: "prompt ID"},
			{Name: "var", Kind: "vars", Description: "variables, key:value (repeatable)"},
			{Name: "onMissing", Kind: "select:error|keep|empty", Description: "missing-variable policy"},
		},
		CLITemplate: "alaws prompt render {path} {id} --var {var} --on-missing {onMissing}",
		GoTemplate:  `book.Prompt({id}).Render(alaws.PromptRenderOptions{...})`,
	},
	{
		ID: "prompt.create", Group: "Prompts", Summary: "Create a new prompt template",
		Method: http.MethodPost, Path: "/api/book/prompts",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "file", Kind: "text", Required: true, Description: "Markdown file path, relative to the book"},
			{Name: "title", Kind: "text", Required: true, Description: "prompt title"},
			{Name: "id", Kind: "id", Required: true, Description: "stable prompt ID"},
			{Name: "after", Kind: "text", Description: "insert after this prompt file"},
		},
		CLITemplate: "alaws prompt create {file} --book {book} --title {title} --id {id} --after {after}",
		GoTemplate:  `alaws.CreatePrompt({book}, {file}, {title}, {id}, alaws.Placement{After: {after}})`,
	},
	{
		ID: "prompt.remove", Group: "Prompts", Summary: "Remove a prompt template",
		Method: http.MethodDelete, Path: "/api/book/prompts",
		Params: []Param{
			{Name: "book", Kind: "book", Required: true, Description: "the book"},
			{Name: "file", Kind: "text", Required: true, Description: "prompt file path"},
		},
		CLITemplate: "alaws prompt remove {book} {id}",
		GoTemplate:  `alaws.RemovePrompt({book}, {file})`,
	},
}

// GET /api/meta/operations
func handleOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, Operations)
}
