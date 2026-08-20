# PromptBook: first-class prompt specifications for AgentLaws

## Context

AgentLaws today models Lawbooks (`alaws.toml` + ordered Markdown sections + numbered laws),
but the actual thing an application sends to an agent — a stitched-together prompt built from
several sections/laws plus runtime variables — has no representation anywhere. It's assembled
ad hoc by application code calling `book.Laws(selector).Render(vars)`, and never appears in the
compiled IR, the web UI, or any export. This means a lawbook's authors can't see, name, version,
or govern the actual prompts built from their laws — only the raw material.

This plan adds **PromptTemplate** as a new first-class object, compiled from the same source
tree, alongside Sections. A PromptTemplate is a named, ID'd, authored Markdown document (just
like a section) whose body can *stitch in* law/section text by reference and leave `{{var}}`
placeholders for values supplied at render time. It gets a "PromptBook" view in the web UI (a
toggle within each LawBook), CLI commands, Go library methods, and export support — reusing the
existing compile → IR → {CLI, library, web UI, HTML/PDF/MD renderers} architecture rather than
building a parallel system.

**Core design decision, stated up front:** stitching law/section text into a prompt is a
**compile-time** operation (deterministic, part of the canonical IR, exactly like law numbering)
— it does *not* resolve `{{var}}` placeholders, which stay **render-time** exactly as `{{var}}`
already works for laws today (§17a in `docs/PLAN1.md`). This means a compiled PromptTemplate's
`Template` field is a plain string with `{{var}}` placeholders still in it, and
`internal/template.Render` (already built, untouched) is reused as-is for the render step. This
mirrors and reuses almost the entire existing linking/rendering/CLI/API architecture instead of
inventing a second one.

**Two other decisions the user gave directly, carried through this whole plan:**
1. **One generic reference notation, not three.** A stitch directive is just `{{ref:<id>}}` — no
   `law:`/`section:`/`prompt:` distinction in the syntax. Whatever the `<id>` resolves to (via
   the existing `resolver.Resolve` precedence chain, now extended with prompts, §6) determines
   how it expands. The author never has to know or say what *kind* of thing they're citing.
2. **Navigation is bidirectional, everywhere.** Today `alaws:` links only go one direction:
   prompt → the law/section it cites. This plan also requires the reverse — from a law's or
   section's own page/anchor (in the UI, HTML export, and PDF export) to every prompt that cites
   it. This needs a small compile-time backlink index (§6a) plus a render-side "used in
   prompts" line wherever a law/section already renders (§10, §11) — not just a UI nicety.

---

## 1. Source format

### `alaws.toml`

```toml
title = "Engineering Governance"
ordering = [...]

promptTemplates = [
  "prompts/code-review.md",
  "prompts/incident-triage.md",
]
```

`promptTemplates` is a flat list (no hierarchy — prompts don't nest like chapters/sections do).
Every listed file must exist (`missing-file` error, mirroring `ordering`); every `.md`/`.mdx`
file not listed is flagged (`unused-file` warning) — `discovery.UnorderedFiles` gets called with
`ordering` and `promptTemplates` entries unioned, so prompt files don't fire false positives.

**Important gotcha to fix while touching this:** `internal/ordering`'s own private `tomlConfig`
(`internal/ordering/ordering.go:46-49`) only declares `Title`/`Ordering`. `go-toml`'s `Marshal`
drops unknown-to-the-struct keys, so any existing `ordering.Insert/Move/Remove/NewBook` call
would **silently delete `promptTemplates` from `alaws.toml`** the moment this ships, unless that
struct also gains `PromptTemplates []string \`toml:"promptTemplates"\`` and every read-modify-write
in that file round-trips it unchanged.

### Prompt Markdown file

Same shape as a section file (frontmatter + two structural regions), reusing the existing
`<!-- alaws:commentary -->` marker plus a new `<!-- alaws:promptTemplate -->` marker:

```md
---
title: Code Review Prompt
id: engineering.prompts.code-review
---

<!-- alaws:commentary -->

Used by the CI review bot before approving a PR touching payment code.

<!-- alaws:promptTemplate -->

You are reviewing a pull request in {{repo}} authored by {{author}}.

Apply the following laws:

{{ref:engineering.coding}}

{{ref:engineering.security.secrets.no-secrets-in-scm}}

Decision must cite law numbers.
```

`title` and `id` are mandatory frontmatter, exactly like sections (`missing-title`/`missing-id`
diagnostics reused). No `level` — prompts are flat, not hierarchical.

### Stitching syntax (new, inside `<!-- alaws:promptTemplate -->` only)

**One generic directive**, `{{ref:<id>}}` — no separate law/section/prompt forms. `<id>` is
resolved with the exact same `resolver.Resolve` precedence chain `alaws:` links already use
(section ID, fully-qualified law identity, citation number, section number, unambiguous bare
slug — now also a prompt ID, §6), so any ID that already resolves for a link resolves here too.
What it expands to depends on what it resolved to, not on how it was written:

| Resolved kind | Expands to |
|---|---|
| Law | that law's `"<Number> <Text>"`, vars left intact |
| Section | that section's laws only (not its commentary — a prompt is agent-facing operative content, matching §16's philosophy; mirrors `Selector{SectionIDs:[ref]}`), one `"<Number> <Text>"` per line |
| Prompt | that prompt's own fully-expanded `Template` (composable; cycle-checked) |

This is unambiguous against plain `{{var}}` because the existing identifier grammar
(`internal/template.go`'s `isIdentByte`) doesn't allow `:` — no grammar change needed there at
all. A directive that fails to resolve is left as literal text (same "visibly broken, not
silently swallowed" rule as `alaws:` links, docs/linking.md §4.1) and reported via a new
`dangling-prompt-reference` diagnostic.

---

## 2. Model changes (`internal/model/model.go`)

```go
type LawbookMetadata struct {
    Title           string
    Ordering        []string
    PromptTemplates []string // new
}

type SegmentKind int
const (
    SegmentText SegmentKind = iota
    SegmentLawRef
    SegmentSectionRef
    SegmentPromptRef
)

// PromptSegment is one piece of a PromptTemplate's body: either literal
// authored text, or a resolved {{ref:x}} reference. Kind records what the
// reference resolved to (law/section/prompt) - the source syntax itself
// doesn't distinguish these (§1), but the renderer still needs to know
// which it got in order to decide how Expanded was built.
type PromptSegment struct {
    Kind      SegmentKind
    Text      string // literal source text, valid iff SegmentText
    RefToken  string // raw token inside {{ref:...}}, valid for ref kinds
    RefAnchor string // resolved stable anchor (section ID, sectionID.slug, or prompt ID)
    RefLabel  string // human label for display (citation number, or section number+title)
    Expanded  string // the stitched-in text, {{var}} left intact, valid for ref kinds
}

type PromptTemplate struct {
    ID                string
    Title             string
    Commentary        string
    Segments          []PromptSegment
    Template          string   // flattened Segments — canonical, hashable/signable, render-ready
    Vars              []string // sorted, deduped {{var}} identifiers found in Template
    ReferencedAnchors []string // sorted, deduped anchors this prompt pulls in, direct + transitive via {{ref:<prompt>}} — the backlink source, and also shown in the UI as "this prompt draws from"
    Source            SourceRef
}

type Lawbook struct {
    Metadata        LawbookMetadata
    Sections        []Section
    Prompts         []PromptTemplate  // new
    PromptBacklinks map[string][]string // new — anchor (section ID / sectionID.slug / prompt ID) -> sorted prompt IDs that reference it, direct or transitive. The reverse index that makes navigation bidirectional (§6a).
    Provenance      Provenance
}
```

`Template` (not `Segments`) is what gets hashed/signed and JSON-exported — it's the deterministic
canonical form, same status as a `Law.Text`. `Segments` is presentation metadata that lets
renderers show either the expanded text or a "just show what's referenced" compact form (§6)
without re-deriving it. `PromptBacklinks` is derived, not authored — computed once in
`promptexpand.Expand` (§6a) from every prompt's `ReferencedAnchors`, and consumed identically by
the web UI, HTML, PDF, and Markdown renderers.

---

## 3. Parsing (`internal/parser/parser.go`)

Add `ParsePromptTemplate(path string) (ParsedPrompt, error)`, factoring the frontmatter-delimiter
scan (`---`...`---`) out of `ParseSection` into a small shared helper both call. `ParsedPrompt`:

```go
type ParsedPrompt struct {
    ID          string
    Title       string
    Commentary  string
    RawTemplate string // unexpanded body, directives intact
    Source      model.SourceRef
}
```

Required marker: `<!-- alaws:commentary -->` then `<!-- alaws:promptTemplate -->` (new constant
`promptTemplateMarker`), same ordering/duplicate rules as `commentaryMarker`/`lawsMarker` today.
Missing frontmatter fields or markers reuse `missing-title`/`missing-id`/`invalid-metadata`;
missing the `promptTemplate` marker is a new `missing-prompt-template` error.

---

## 4. Directive expansion — new package `internal/promptexpand`

A new package (same tier as `internal/numbering`/`internal/resolver` — imported by the compiler,
imports only `model`/`resolver`/`parser`/`template`, no cycle) that turns
`[]parser.ParsedPrompt` + the already-numbered `model.Lawbook` into `[]model.PromptTemplate`:

```go
func Expand(book model.Lawbook, raw []parser.ParsedPrompt) ([]model.PromptTemplate, []validator.Diagnostic)
```

Algorithm: regex-scan each prompt's `RawTemplate` for `\{\{ref:([^}]+)\}\}`, splitting into
`PromptSegment`s. Each match resolves once via `resolver.Resolve(book, token)` — the *same*
generic call an `alaws:` link uses — and branches on the returned `Kind`:
- `KindLaw`/`KindSection` resolve directly against `book` (no new resolution logic beyond what
  `resolver.Resolve`/looking up that section's `Laws` already gives).
- `KindPrompt` resolves recursively into the sibling prompt with a per-branch "visiting" set for
  cycle detection (standard DFS-with-recursion-stack), memoized so a prompt referenced by two
  others is only expanded once. A cycle produces `circular-prompt-reference` (error).

An unresolved token produces `dangling-prompt-reference` (error) and the directive is left as
literal text.

After all segments for a prompt are resolved: `Template` = concatenation of every segment's
`Text`/`Expanded`. Run the *existing, unchanged* `template.ValidateSyntax(Template)` on the
result (this is what catches malformed `{{var}}` — and, as a side effect, catches a directive
whose id never resolved and so wasn't replaced, since a leftover `ref:...` token isn't a valid
bare identifier either). Compute `Vars` via one small new addition to `internal/template`:

```go
// internal/template/template.go
func Vars(text string) []string // sorted, deduped {{identifier}} names — reuses ValidateSyntax's walk
```

Also compute `ReferencedAnchors`: the set of every `RefAnchor` this prompt's segments touch
directly, unioned with the *already-computed* `ReferencedAnchors` of every `KindPrompt` segment
it includes (available for free since those are resolved — and memoized — before the current
prompt finishes, by the same DFS). This makes a prompt-of-a-prompt's transitive reach correct
with no extra pass: if A includes B and B includes law L, L ends up in both B's and A's
`ReferencedAnchors`.

---

## 5. Compiler pipeline (`internal/compiler/compiler.go`)

Insert prompt parsing + expansion between building `lawbook` (after `numbering.Assign`) and
`validator.Validate(lawbook)`, so prompts see final law numbers/text and the validator sees
`lawbook.Prompts` too:

```go
// same missing-file/invalid-metadata loop shape as the existing ordering loop, over meta.PromptTemplates
var parsedPrompts []parser.ParsedPrompt
// ...

lawbook := model.Lawbook{Metadata: meta, Sections: numbered}
prompts, promptDiags := promptexpand.Expand(lawbook, parsedPrompts)
lawbook.Prompts = prompts
diags = append(diags, promptDiags...)

diags = append(diags, validator.Validate(lawbook)...) // now also checks Prompts (below)
```

`discovery.UnorderedFiles` call site: pass `append(meta.Ordering, meta.PromptTemplates...)`
instead of `meta.Ordering` alone.

### Validator additions (`internal/validator/validator.go`)

Extend the existing `Validate(book model.Lawbook)` — it already builds a `sectionIDs`/`seen` map
for `duplicate-id`; extend that same map so **prompt IDs share the section-ID namespace** (a
prompt ID colliding with a section ID is `duplicate-id`, same as two sections colliding — this is
what lets `alaws:` links and the resolver treat prompts as "just another addressable thing"
with zero new namespace-juggling code, §6). New codes:

| Code | Severity | Condition |
|---|---|---|
| `missing-prompt-template` | Error | prompt file has no `promptTemplate` marker |
| `empty-prompt-template` | Warning | promptTemplate region has no content |
| `dangling-prompt-reference` | Error | a `{{ref:x}}` directive didn't resolve |
| `circular-prompt-reference` | Error | a `{{ref:x}}` chain (through prompt-to-prompt refs) cycles back to itself |

---

## 6. Resolver / linking reuse (`internal/resolver/resolver.go`)

Add `KindPrompt` and a `Prompt model.PromptTemplate` field on `Resolved`. Extend `Resolve`'s
step (a) (exact-ID match) to also check `book.Prompts` — prompts share the global ID namespace
with sections (validated above), so this is a same-shaped loop, not new precedence logic. Extend
`AnchorFor` with `case KindPrompt: return r.Prompt.ID`.

This is the single change that makes `[link](alaws:engineering.prompts.code-review)` work
**everywhere `alaws:` links already work** — HTML, PDF (internal link + anchor registration),
Markdown export, and the web UI's hash router — with **zero changes** to `RenderFragment`, the
goldmark AST-walk transform, or the PDF anchor-sentinel mechanism (`docs/linking.md` §4.1–§4.5).
That machinery only ever calls `resolver.Resolve`/`resolver.AnchorFor`, which now just knows one
more `Kind`.

### Display-mode helper (new, small — put in `internal/resolver` alongside `AnchorFor`)

```go
// PromptDisplayText renders p's segments as Markdown, either fully expanded
// (ref segments show their stitched-in text) or compact (ref segments show
// an `alaws:` link to the original law/section/prompt instead). Used by
// every renderer (§9) and the web UI (§8) so the "expanded vs IDs" choice
// has exactly one implementation.
func PromptDisplayText(p model.PromptTemplate, expanded bool) string
```

Compact mode's link label uses `RefLabel` (citation number, or section number+title); href is
`alaws:` + `RefAnchor`, resolved through the exact same `RenderFragment` pipeline as everything
else once the caller feeds this Markdown string through it.

### Backlink index — the reverse direction (`internal/promptexpand`)

`alaws:`/`{{ref:}}` links only ever point *from* a prompt *to* a law/section/prompt. The user
explicitly asked for the reverse too: standing on a law's or section's own page/anchor, see which
prompt(s) cite it, in the UI **and** in every export format — not just as a UI nicety.

Since every prompt's `ReferencedAnchors` (§4) is already computed at compile time, the backlink
index is one small pass at the end of `promptexpand.Expand`, after every prompt is resolved:

```go
backlinks := map[string][]string{} // anchor -> prompt IDs
for _, p := range prompts {
    for _, anchor := range p.ReferencedAnchors {
        backlinks[anchor] = append(backlinks[anchor], p.ID)
    }
}
// sort + dedupe each entry
```

This becomes `Lawbook.PromptBacklinks` (§2). Nothing needs `resolver.Resolve` again here — it's
a pure aggregation over already-resolved data. Every consumer (web UI §10, HTML/PDF/MD §11) reads
this one map to render a "used in prompts: …" line under a law or section, linking to
`alaws:<promptID>` — which already resolves and renders correctly everywhere, because `KindPrompt`
was added to `resolver.Resolve`/`AnchorFor` above. No second link mechanism, just the existing one
consumed in the other direction.

---

## 7. Go library (`pkg/alaws/prompts.go`, new file — mirrors `laws.go`)

```go
func (b *Book) Prompts() []model.PromptTemplate
func (b *Book) Prompt(id string) (Prompt, error) // wraps model.PromptTemplate, adds Render — same pattern as LawSet wrapping []model.Law

type Prompt struct{ model.PromptTemplate }

type PromptRenderOptions struct {
    Vars      map[string]string
    OnMissing MissingPolicy // reuses the existing alias from laws.go
}

func (p Prompt) Render(opts PromptRenderOptions) (string, error) {
    return template.Render(p.Template, opts.Vars, opts.OnMissing) // internal/template, unchanged
}
```

Example (goes in `pkg/alaws/example_test.go` per `AGENTS.md`'s godoc conventions, and is what
the web UI's per-prompt "Go usage" panel shows, §8):

```go
book, _ := alaws.Load("./engineering")
prompt, _ := book.Prompt("engineering.prompts.code-review")
fmt.Println(prompt.Vars) // ["author", "repo"]
text, _ := prompt.Render(alaws.PromptRenderOptions{
    Vars: map[string]string{"repo": "org/app", "author": "ci-bot"},
})
```

Also add to `pkg/alaws/ordering.go`-equivalent mutation surface (new functions in
`internal/ordering`, extended `tomlConfig` per §1's gotcha, flat-list Insert/Move/Remove
mirroring the existing section ones but without subtree logic):

```go
func CreatePrompt(book, file, title, id string, placement Placement) error
func RemovePrompt(book, id string) error
func MovePrompt(book, id string, placement Placement) error
```

---

## 8. CLI (`internal/cli/promptbook.go`, new file — **not** `prompt.go`, which already holds
unrelated interactive-terminal helpers `isInteractive`/`promptChoice`)

Mirrors `alaws section`/`alaws law`/`alaws render` exactly:

```
alaws prompt create <book> <file> --title "..." --id "..." [--after <id>|--position N]
alaws prompt list <book> [--json]
alaws prompt show <book> <id> [--json] [--raw]      # --raw = segments with directives unexpanded
alaws prompt remove <book> <id> [--force]
alaws prompt vars <book> <id> [--json]              # what needs filling
alaws prompt render <book> <id> [--var k=v]... [--vars-file f] [--on-missing error|keep|empty] [--json]
```

Same cross-cutting rules as every other command (`--json`, exit codes, `--dry-run`,
`--root`) per `docs/PLAN1.md` §32 — no new conventions.

---

## 9. Server API (`internal/server/api.go`) + Operations registry

- Extend `handleCompile`'s response with `prompts: []model.PromptTemplate` and a parallel
  `renderedPrompts: map[string]RenderedPrompt{CommentaryHTML, TemplateHTML, CompactHTML}` —
  built the same way `RenderedSections` builds `RenderedSection` today (`pkg/alaws/render.go`),
  reusing `renderhtml.RenderFragment` + the `resolve` closure + the `__BOOK_PATH__`
  placeholder-replace trick already in `handleCompile`. `TemplateHTML` comes from
  `resolver.PromptDisplayText(p, true)`, `CompactHTML` from `PromptDisplayText(p, false)`, both
  piped through `RenderFragment`.
- New `GET /api/book/prompt/render?path=&id=&var=key:value&onMissing=` — mirrors `handleRender`
  (§ existing `handleRender` at `internal/server/api.go:270`), calling `Book.Prompt(id).Render(...)`.
- New POST/DELETE on a `/api/book/prompts` endpoint mirroring `handleChapters`/`handleSections`
  for create/remove, and reuse `handleMove`'s shape (`kind: "prompt"`) for reordering.
- **No new endpoint needed for backlinks.** `b.Lawbook()` already includes `PromptBacklinks`
  (§2/§6a) once the model carries it, and `handleCompile` already returns the full `lawbook` —
  so the existing `/api/book/compile` response carries the reverse index for free; the UI reads
  `lawbook.PromptBacklinks[anchor]` directly (§10).
- Add corresponding entries to `Operations` in `internal/server/operations.go` (`prompt.list`,
  `prompt.render`, `prompt.create`, ...) — this is the *existing* mechanism
  (`GoTemplate`/`CLITemplate` per operation, served at `/api/meta/operations`, consumed by
  `Playground.tsx`) that already answers "how do I call this from the CLI/Go library" for every
  other capability; prompts get the same generic treatment there for free.

---

## 10. Web UI — PromptBook toggle (`web/src/views/BookDetail.tsx`, `web/src/router.ts`, `web/src/api.ts`)

### Router (`router.ts`)

The hash scheme already special-cases a literal third segment (`playground`, `router.ts:18`).
Add the same shape for prompts: `#/books/<path>/prompts` (list) and
`#/books/<path>/prompts/<promptID>` (detail) — unambiguous because no real section ID is
literally `"prompts"`, exactly like the existing `playground` special-case. `Route` gains:

```ts
export type Route =
  | { name: "books" }
  | { name: "book"; path: string; section?: string; law?: string }
  | { name: "prompts"; path: string; id?: string } // new
  | { name: "playground"; path: string };
```

### `api.ts`

New types (`PromptTemplate`, `PromptSegment`, `RenderedPrompt`) mirroring `Law`/`Section`/
`RenderedSection` already there, plus client functions: `compile()`'s `CompileResult` gains
`prompts`/`renderedPrompts` fields (no new call — same endpoint), and new `promptRender()`,
`createPrompt()`, `removePrompt()`, `movePrompt()` following the exact `req()`/`qs()` pattern
every other function in the file already uses.

### `BookDetail.tsx` (or a new sibling `PromptBookDetail.tsx` reusing its shell)

Add a two-way toggle in the sidebar header — "Laws" / "Prompts" — persisted via
`localStorage` exactly like the existing `sidebarVisible`/`sidebarWidth` state
(`BookDetail.tsx:66-70`). This is the "PromptBook toggle within each LawBook" from the request:
same book, same URL family, a mode switch, not a separate page.

**In Prompts mode:**
- Sidebar lists `Lawbook.Prompts` (flat, `promptTemplates` order) the same way the sidebar today
  lists the section tree (`buildTree`, `BookDetail.tsx:28-41` — trivial here since prompts have
  no hierarchy, just a flat `<ul>`).
- Selecting a prompt shows: title, ID, rendered `CommentaryHTML`; a segmented view of the
  template — literal text rendered normally, each ref segment shown in a bordered/labelled block
  ("included from `engineering.coding`") with a **jump-to-source** link. That link is exactly
  `navigate({ name: "book", path, section: sectionID, law: lawSlug })` — the same route the
  existing `alaws:` link-click already lands on (`router.ts` + `BookDetail.tsx:128-146`'s
  scroll/flash effect need no changes; this is literally reusing the existing law/section
  detail view, just triggered from the Prompts sidebar instead of an inline `alaws:` link).
- A **Variables** panel: one input per `PromptTemplate.Vars` entry, local component state.
- A **Preview** panel: calls the new `api.promptRender(path, id, { vars, onMissing })` on every
  keystroke (debounced) and shows the returned text — this is the live "how it'd look" preview,
  computed server-side so there is exactly one implementation of `{{var}}` substitution
  (`internal/template.Render`), not a duplicated TS reimplementation.
- A **Go usage** panel: a static code block generated client-side from the prompt's own `ID` and
  `Vars` (no server call needed — just string-templating the pattern from §7's example with the
  actual id/var names substituted in), with the same copy-button pattern already used for
  `copiedPath`/`copiedLawLink` (`BookDetail.tsx:58-59`).
- A **display-mode** toggle (Expanded / Compact) for the template view itself, using
  `TemplateHTML`/`CompactHTML` from `renderedPrompts` (§9) — lets an author preview what the
  "IDs only" export mode (§11) will actually look like.

**In Laws mode (the existing section/law detail view), required, not optional:** wherever a
section header or an individual law already renders (`BookDetail.tsx`'s existing section/law
markup), if `lawbook.PromptBacklinks[anchor]` is non-empty, render a small "Used in prompts:"
line listing each prompt's title as a link. Clicking it switches to Prompts mode and navigates to
that prompt (`navigate({ name: "prompts", path, id: promptID })`) — the return trip for the
jump-to-source link two bullets up. This is the UI half of the bidirectional-navigation
requirement; §11 is the export half.

---

## 11. Export

New shared option, threaded through every renderer:

```go
type PromptDisplayMode int
const (
    PromptDisplayOff      PromptDisplayMode = iota // no PromptBook section at all
    PromptDisplayExpanded                           // default when book.Prompts is non-empty
    PromptDisplayCompact                            // {{ref:x}} shown as an alaws: link, not inlined text
)
```

**Sane default:** auto — `PromptDisplayOff` if `len(book.Prompts) == 0` (today's exact behavior,
zero change for existing books with no `promptTemplates` key), else `PromptDisplayExpanded`.
`{{var}}` placeholders are **never** resolved in an export — export is a compiled-artifact view,
not a render instance, exactly like today's HTML/PDF/MD export shows `Number`+`Text` with
`{{var}}` still in it for laws (§17a already establishes "resolution happens only at the
extraction/render boundary" — export isn't that boundary).

Each renderer (`internal/renderer/html`, `/pdf`, `/markdown`) gets a `renderPrompts(w, book,
resolve, mode)` sibling to the existing `renderSections` (html.go:170), using
`resolver.PromptDisplayText` (§6) to pick expanded-vs-compact text, then the exact same
`RenderFragment` call every section/law already goes through — **no new Markdown-handling code
in any renderer.** Each prompt renders under its own heading with `id=` the prompt's anchor
(same mechanism as section headings, `html.go:174`), so:
- HTML/Markdown: works with zero new anchor-registration code (raw `<h*>`/`<a id>` as today).
- PDF: also zero new anchor-registration code for the *prompt's own* anchor, since heading-id
  registration already exists in the vendored library (per `docs/linking.md` §4.3). The only
  *new* PDF work is emitting the `<!--alaws-anchor:...-->` sentinel (already-built mechanism,
  §4.3) for law-citation lines that appear *inside* an expanded prompt body, so a reader clicking
  a link *into* the middle of a prompt lands correctly — reuses, doesn't extend, that mechanism.

**Reverse links (required in every export, not just the UI):** `renderSections` (the existing
function that already writes each section heading and each law `<li>`/paragraph, §6/html.go:170)
gains one small addition: after writing a section's or law's own content, if
`lawbook.PromptBacklinks[anchor]` is non-empty, emit a "Used in prompts:" line whose links are
`alaws:<promptID>` — the exact same link syntax `{{ref:x}}`/`alaws:` links already use, so it
needs no new resolution or anchor mechanism in any format (HTML, PDF, Markdown): `KindPrompt` was
already added to `resolver.Resolve`/`AnchorFor` in §6, and the PDF anchor for the *target* prompt
heading already exists because prompts render as headings (previous paragraph). This is what
completes the round trip — a reader can land on a law from a search or citation, see which
prompts govern it, and click through, in every export format.

CLI/API/UI surface:
- `alaws compile`/`alaws export` gain `--prompts=auto|on|off` and `--prompts-display=expanded|compact`
  (defaults `auto`/`expanded`, matching the library default above).
- `handleExport`/`handleExportAll` (`internal/server/api.go`) gain the same two query params.
- The web UI's existing export menu (`showExportMenu`, `BookDetail.tsx:64`) gains a checkbox
  ("Include PromptBook") and a select ("Expanded / Compact"), pre-filled from the sane default,
  wired into `api.exportURL`/`exportAllURL`'s query string (`api.ts:214,224`).

---

## 12. Diagnostics summary (new codes)

`missing-prompt-template`, `empty-prompt-template`, `dangling-prompt-reference`,
`circular-prompt-reference` (all documented in `internal/validator/validator.go`'s `Diagnostic`
doc comment, alongside the existing code list at `validator.go:34-38`). `missing-title`/
`missing-id`/`invalid-metadata`/`missing-file`/`unused-file`/`duplicate-id`/`invalid-template`
are reused as-is for prompts.

---

## 13. Testing

Follow `AGENTS.md`'s existing conventions exactly (table-driven, stdlib-only, one test per
diagnostic code, golden fixtures):
- `internal/parser`: `ParsePromptTemplate` — valid, missing marker, missing id/title.
- `internal/promptexpand`: one test per resolved kind behind `{{ref:x}}` (law by id/slug/citation,
  section, nested prompt, unresolved → dangling, cyclic → circular), `Vars` extraction
  correctness, and `ReferencedAnchors`/`PromptBacklinks` correctness — including the transitive
  case (prompt A refs prompt B refs law L: assert L's backlink list contains both A and B).
- `internal/template`: `Vars()` — new, small, table-driven like `ValidateSyntax`'s existing tests.
- `internal/validator`: one test per new code, mirroring existing style.
- `internal/resolver`: `Resolve` now also matches prompt IDs; `AnchorFor` `KindPrompt` case;
  `PromptDisplayText` expanded vs. compact.
- `internal/ordering`: extend existing tests to assert `promptTemplates` round-trips through
  `Insert`/`Move`/`Remove`/`NewBook` unchanged (the §1 gotcha) even when those functions don't
  touch it.
- New fixture: `fixtures/basic` (or a new `fixtures/prompts`) gets a `prompts/` dir +
  `promptTemplates` entry, exercised by a compiler golden test.
- End to end: `make test`; manual browser check (`make serve`) of the Prompts toggle, the
  jump-to-source links, variable fill-in + live preview, export with both display modes, **and
  both navigation directions**: from a prompt to a cited law/section, and from that law's/
  section's own page back to the prompt(s) citing it — in the UI, the HTML export, and the PDF
  export — per `AGENTS.md`'s "don't test the web UI with Go tests" rule.

---

## 14. Suggested phased delivery

1. **Model/parse/compile/validate** (§1-§5) — `promptTemplates` compiles into `Lawbook.Prompts`,
   including `ReferencedAnchors`/`PromptBacklinks`, with no consumer yet; `alaws validate`/
   `alaws compile --format json` already show all of it.
2. **Resolver + Go library + CLI** (§6-§8) — `alaws prompt render` works end to end; `alaws:`
   links to prompts work in the existing HTML/PDF/MD exports with zero display-mode work yet
   (always expanded, since that's what `Template` already is). Reverse "used in prompts" links
   under laws/sections (§6a) can ship in this phase too, since they only need `PromptBacklinks`
   and the existing `alaws:` link mechanism — no display-mode work depends on them.
3. **Web UI PromptBook toggle** (§10) + the new server endpoints (§9), including both navigation
   directions in the UI.
4. **Export display modes** (§11) — compact mode, CLI/API/UI flags, sane defaults.

Each phase is independently mergeable and testable, and later phases only add consumers of the
IR built in phase 1 — nothing in phases 2-4 requires revisiting the model.



----

Implementation Update

The PromptBook feature is now fully implemented end-to-end. Here's a summary of what was built:
Summary of Changes
Phase 1: Model/Parse/Compile/Validate
- internal/model/model.go: Added PromptTemplate, PromptSegment, SegmentKind, PromptBacklinks to the Lawbook IR
- internal/parser/parser.go: Added ParsePromptTemplate, promptTemplateMarker, extended tomlConfig for promptTemplates
- internal/ordering/ordering.go: Extended tomlConfig to round-trip promptTemplates through all mutations
- internal/template/template.go: Added Vars() function to extract {{identifier}} names
- internal/promptexpand/: New package for {{ref:x}} directive expansion with cycle detection, memoization, and backlink computation
- internal/validator/validator.go: Added prompt validation (duplicate-id, invalid-template)
- internal/compiler/compiler.go: Integrated prompt parsing into the compilation pipeline
- fixtures/basic/: Added prompts/code-review.md fixture
Phase 2: Resolver + Go Library + CLI
- internal/resolver/resolver.go: Added KindPrompt, extended Resolve/AnchorFor, added PromptDisplayText and ResolvePrompt
- pkg/alaws/prompts.go: New file with Book.Prompts(), Book.Prompt(), Prompt.Render()
- pkg/alaws/ordering.go: Added CreatePrompt, RemovePrompt, MovePrompt
- internal/cli/promptbook.go: New alaws prompt commands (create, list, show, vars, render, remove, move)
Phase 3: Server API + Web UI
- internal/server/api.go: Extended handleCompile with renderedPrompts, added /api/book/prompt/render and /api/book/prompts endpoints
- internal/server/operations.go: Added prompt operations to the registry
- web/src/router.ts: Added prompts route (#/books/<path>/prompts/<id>)
- web/src/api.ts: Added PromptTemplate, RenderedPrompt types and promptRender, createPrompt, removePrompt client functions
- web/src/views/BookDetail.tsx: Added PromptBook toggle (Laws/Prompts mode), prompt list sidebar, prompt detail view with template display, variables panel, live preview, Go usage panel, references, and bidirectional backlinks
- web/src/app.css: Added CSS for mode toggle, prompt template sections, variables, preview, backlinks
Phase 4: Export
- internal/renderer/html/html.go: Added renderPrompts and backlink rendering in sections/laws
- internal/renderer/markdown/markdown.go: Added renderPrompts and backlink rendering
- internal/renderer/pdf/pdf.go: Added PromptBook section and backlink rendering in buildMarkdownInto
Key Features
- Stitching: {{ref:<id>}} resolves laws, sections, or other prompts at compile time
- Variables: {{var}} placeholders are left for render-time substitution
- Bidirectional navigation: From prompt → cited law/section, and from law/section → prompts that cite it
- Cycle detection: Circular prompt references are caught with clear error messages
- Transitive backlinks: If prompt A includes prompt B which cites law L, L's backlinks include both A and B
- Web UI: Laws/Prompts toggle, template preview (expanded/compact), variable fill-in with live preview, Go usage code generation
You can test it by running make serve and navigating to the fixture book.
