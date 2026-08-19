# Stable cross-referencing for individual laws

## 0. What this document is

This is the complete implementation spec for adding stable, cross-file
cross-references to individual laws in AgentLaws. It is written to be
self-contained: a reader with no prior context on this design discussion
should be able to execute it end to end from this document alone.

**Step 0 of execution, before any other change**: copy this document
verbatim to `docs/linking.md` in the repository (`git add docs/linking.md`).
That file is the durable spec of record for this feature; this plan file is
disposable. Everything below refers to itself as "this document" so the copy
reads correctly in its new home.

Everything in this document targets the AgentLaws Go module at
`github.com/shrsv/AgentLaws`, repo root `/home/shrsv/bin/AgentLaws`.

---

## 1. Context and rationale

AgentLaws already solves file/section-level linking well: every section has
an author-assigned, stable, dotted `id` in its frontmatter (e.g.
`engineering.security.secrets`), independent of presentation position
(`internal/model/model.go:27-31`; author-set via `id:` YAML frontmatter,
parsed in `internal/parser/parser.go:81-85,110-119`). What's missing is the
same stability one level down, for individual laws (numbered clauses).

Today a law's only identifier is `Number` (e.g. `"2.5.3"`), recomputed from
list position on every compile by `internal/numbering/numbering.go`. The code
already documents this as unsafe:

```go
// internal/model/model.go:14-18
// Law is a single numbered clause within a section's laws region.
//
// Number is the canonical citation (e.g. "2.5.3"), assigned during
// compilation. It is never authored directly and must not be treated as a
// stable identity — SectionID + Index is the stable identity (§14).
```

Yet `Number` is the *only* thing `resolver.ResolveLaw`
(`internal/resolver/resolver.go`), the HTML renderer's `id=` attribute
(`internal/renderer/html/html.go:129-130`), and the web UI's law-history
lookup (`web/src/views/BookDetail.tsx`, keyed by `law.Number`) key on today —
so inserting or removing one law anywhere above another silently breaks every
existing reference to everything below it.

**Content-addressing was considered and rejected.** Hashing a law's text as
its identity is strictly worse than positional numbers: a pure typo fix would
churn the hash even though the law's *identity* (what it's regulating) hasn't
changed at all, whereas positional numbers at least survive pure text edits
untouched. Identity has to be something an author assigns and controls, not
something derived from content or position.

**Precedent.** This design mirrors Akoma Ntoso (the OASIS/legal-document XML
standard), which splits every provision's address into `@eId` (hierarchical,
positional, "non-persistent") and `@wId` (assigned once, "will never change
once assigned" — persistent). That is exactly the existing `Number` vs `ID`
split already in place for *sections*; this feature extends it down to
*laws*. The plain-text syntax for minting and referencing such an id follows
pandoc/kramdown's `{#id}` inline attribute list (for defining an anchor) and
AsciiDoc's `[[anchor]]` / `<<xref>>` (for referencing one, resolved at build
time into the right link mechanics per output format — HTML anchor, PDF
internal link, etc.), which is exactly the shape of problem this feature
needs to solve across HTML, PDF, Markdown, JSON, and the web UI.

---

## 2. The addressing model (core design)

A law's **fully-qualified identity** is:

```
<section-id>.<law-slug>
```

e.g. `engineering.security.secrets.no-secrets-in-scm`, where
`engineering.security.secrets` is that law's section's existing, already
stable `Section.ID`, and `no-secrets-in-scm` is a new, author-assigned
per-law slug.

This is deliberately visually indistinguishable from "one more level of
section nesting" — it reads as a natural extension of the dotted-namespace
convention section IDs already use (README: "Namespaces are encouraged:
`engineering.security.credentials`"). A law's slug therefore only needs to be
**unique within its own section** (not lawbook-wide): global uniqueness of
the fully-qualified identity follows automatically, because section IDs are
already required unique lawbook-wide (existing `duplicate-id` diagnostic,
`internal/validator/validator.go:87-89`).

This fully-qualified form is what makes cross-file linking work: a law
defined in `security/secrets.md` can be linked to by its identity from
`coding.md`, or from any other file in the *same lawbook* (one directory with
one `alaws.toml`) — resolution is scoped to one compiled `model.Lawbook`,
matching how `resolver.Resolve`/`ResolveLaw`/`ResolveSection` already operate
on one `model.Lawbook` value. Cross-*lawbook* (i.e. across two independent
`alaws.toml` roots) resolution is out of scope for this feature; note it as a
future extension (e.g. a book-id prefix) but do not build it now.

A **bare slug** (just `no-secrets-in-scm`, no section prefix) is also
resolvable as a convenience, but only when it is unambiguous — i.e. exactly
one law lawbook-wide carries that slug. If two sections both happen to use
the same slug for different laws, the bare form is ambiguous and only the
fully-qualified form resolves; this is fine and expected, not an error, since
per-section slug uniqueness (not lawbook-wide) is all that's required.

### 2.1 Full resolution algorithm

This replaces and formalizes the ad hoc double-try already informally
present in the CLI's `show` command (`internal/cli/compile.go:123-152`, which
today tries `Book.Resolve` (citation) then falls back to `Book.Section`
(id-or-number)). Implement one function:

```go
// internal/resolver/resolver.go

// Kind distinguishes what a resolved reference points at.
type Kind int

const (
	KindLaw Kind = iota
	KindSection
)

// Resolved is the result of resolving one reference token.
type Resolved struct {
	Kind    Kind
	Law     model.Law     // valid iff Kind == KindLaw
	Section model.Section // valid iff Kind == KindSection
}

// Resolve resolves token against book, trying — in this exact order — every
// addressing form AgentLaws supports. See docs/linking.md §2.1 for the full
// specification and the rationale for this precedence order.
func Resolve(book model.Lawbook, token string) (Resolved, error) {
	// (a) Exact Section.ID match — highest precedence. A bare dotted id
	// always means "this section", even if, pathologically, some other
	// section's id + a law slug would also produce the same string (see
	// the "ambiguous-identity" validator diagnostic in §5, which flags
	// this case at compile time so it's never a silent surprise).
	for _, s := range book.Sections {
		if s.ID == token {
			return Resolved{Kind: KindSection, Section: s}, nil
		}
	}

	// (b) Fully-qualified law identity: "<section-id>.<law-slug>". Split at
	// the LAST '.' — section ids may themselves contain multiple dots, but
	// a law slug (see §3 charset) never contains a '.', so the last dot is
	// always the correct split point.
	if lastDot := strings.LastIndex(token, "."); lastDot != -1 {
		sectionPart, slugPart := token[:lastDot], token[lastDot+1:]
		for _, s := range book.Sections {
			if s.ID != sectionPart {
				continue
			}
			for _, l := range s.Laws {
				if l.Slug == slugPart {
					return Resolved{Kind: KindLaw, Law: l}, nil
				}
			}
		}
	}

	// (c) Law citation number, e.g. "2.5.3" — legacy/as-compiled form.
	// Still resolvable so an agent's decision log ("Laws: 2.5.1") remains
	// resolvable against the exact compile that produced it, even though
	// this form is not reorder-stable.
	for _, s := range book.Sections {
		for _, l := range s.Laws {
			if l.Number == token {
				return Resolved{Kind: KindLaw, Law: l}, nil
			}
		}
	}

	// (d) Section presentation number, e.g. "2.5".
	for _, s := range book.Sections {
		if s.Number == token {
			return Resolved{Kind: KindSection, Section: s}, nil
		}
	}

	// (e) Bare law slug, unqualified — only if unambiguous lawbook-wide.
	var match *model.Law
	ambiguous := false
	for _, s := range book.Sections {
		for i, l := range s.Laws {
			if l.Slug == "" || l.Slug != token {
				continue
			}
			if match != nil {
				ambiguous = true
			}
			match = &s.Laws[i]
		}
	}
	if match != nil && !ambiguous {
		return Resolved{Kind: KindLaw, Law: *match}, nil
	}
	if ambiguous {
		return Resolved{}, fmt.Errorf("%w: %q is an ambiguous bare slug (used in more than one section) — use the fully-qualified <section-id>.<slug> form", ErrNotFound, token)
	}

	return Resolved{}, fmt.Errorf("%w: %q", ErrNotFound, token)
}
```

Notes for whoever implements this:
- There is no real lexical collision risk between forms (b)/(c)/(e): law
  slugs must start with a lowercase letter (§3 charset), citation numbers are
  digits-and-dots only. The ordering above is about *precedence*, not about
  resolving ambiguity between charsets.
- Keep the existing `ResolveLaw(book, citation string) (model.Law, error)`
  and `ResolveSection(book, id string) (model.Section, error)` functions
  working exactly as they do today (do not remove or change their existing
  behavior/signatures — other code depends on them, see §6 file list). Add
  `Resolve` as a new function alongside them. `ResolveLaw` should internally
  gain the ability to match `l.Slug == token` in addition to its existing
  `l.Number == token` check (this is forms (c) alone, i.e. `ResolveLaw` stays
  a "resolve within `Laws`, by any of its own identifiers" helper); `Resolve`
  is the new top-level entry point implementing the full precedence chain
  above, and should internally call `ResolveLaw`/`ResolveSection` where
  convenient rather than duplicating their loops.

---

## 3. Syntax specification

### 3.1 Minting a law's slug (author-facing Markdown syntax)

Confirmed with the user: the slug attribute is written as `{#slug}`,
either trailing on a one-line law's own line, or alone on its own line
(any indentation) immediately after a multi-line/fenced law's content:

```md
1. Credentials must never be committed to source control. {#no-secrets-in-scm}

2. Run this check before merging:
   ```bash
   make test
   ```
   {#pre-merge-check}
```

**Why one rule handles both placements.** `internal/parser/parser.go`'s
`parseLawLines` (parser.go:167-204) already folds every continuation line of
a clause — including, in the second example, the closing fence line and the
`{#pre-merge-check}` line after it — into one `Text` string via
`lawFencer.fold` (parser.go:229-247): non-fence lines are joined with a
leading space, fence lines are joined with a leading `\n`. By the time a
clause is finalized (`current.Text = strings.TrimSpace(current.Text)`, at
parser.go:185 and parser.go:200), `Text` always ends in `... {#slug}`
regardless of which of the two placements the author used. So exactly one
regex, applied at those two finalization points, handles both:

```go
var slugAttrRe = regexp.MustCompile(`\s*\{#([a-z][a-z0-9-]*)\}\s*$`)
```

Slug charset: `^[a-z][a-z0-9-]*$` — lowercase letters, digits, hyphens,
must start with a lowercase letter. No dots (dots are the section/law
boundary marker in the fully-qualified identity — see §2). No underscores or
uppercase, to keep it a safe, unescaped literal in an HTML `id` attribute, a
PDF named destination, and a URL fragment.

Extraction rule: if `slugAttrRe` matches the end of a clause's fully-folded
`Text`, capture group 1 as the slug, and set `Text` to the string with the
matched suffix removed and re-trimmed. If it does not match, the law has no
slug (its trailing text is left completely untouched — this matters for the
negative test case in §7: a law whose real prose happens to end in something
that merely looks brace-shaped but doesn't match the strict charset, e.g.
ending in `{full stop}`, must NOT be treated as having a slug and must keep
that text verbatim).

### 3.2 Referencing a law or section (author-facing Markdown syntax)

Standard Markdown link syntax with a custom URI scheme:

```md
[see the secrets law](alaws:engineering.security.secrets.no-secrets-in-scm)
[see the security chapter](alaws:engineering.security)
[see it](alaws:no-secrets-in-scm)          <!-- bare slug, only if unambiguous -->
[as originally cited](alaws:2.5.1)         <!-- citation number, legacy form -->
```

This renders as a normal (if inert, unstyled-until-rendered) link in any
plain Markdown viewer, including GitHub — it degrades gracefully exactly the
way a `mailto:` link does. AgentLaws' own renderers recognize the `alaws:`
scheme at render time and rewrite the `href` into whatever the target output
format needs (in-page anchor for HTML, internal PDF link, raw HTML anchor
target for Markdown export, or an app hash-route for the web UI) — see §4.

The token after `alaws:` is resolved with exactly the `Resolve` algorithm in
§2.1 — same precedence order, same forms, no separate parsing rules for
links vs. citations.

---

## 4. Rendering: making links and anchors work in every output format

### 4.1 Shared mechanism

`internal/renderer/html.RenderFragment` (`internal/renderer/html/html.go:22-47`)
currently converts one Markdown string to HTML in isolation, with no
knowledge of the rest of the lawbook. It's called from two places, and both
already have the *full* `model.Lawbook` in scope at the call site:

1. The static exporters, `Render`/`RenderAll` in the same file, which loop
   over every section while assembling one document.
2. `pkg/alaws/render.go:49-68` (`RenderedSections`), which is what
   `internal/server/api.go`'s `handleCompile` calls to build the JSON the
   web UI's `BookDetail.tsx` renders via `dangerouslySetInnerHTML`.

Change `RenderFragment`'s signature to accept a resolver callback:

```go
// resolve, given an alaws: link token, returns the href to use in the
// rendered output, or ok=false if the token didn't resolve (in which case
// the original alaws:token href is left untouched, so a broken link is
// visibly broken rather than silently swallowed — the validator diagnostic
// in §5 is what should catch this before it ships, not silent fallback).
func RenderFragment(md string, resolve func(token string) (href string, ok bool)) (string, error)
```

Implementation: goldmark visits link nodes during parsing/rendering. Add a
small AST-walk step (or a custom goldmark `ast.Transformer` registered on the
shared `markdown` goldmark instance, html.go:26-33) that, for every
`*ast.Link` node whose `Destination` starts with `alaws:`, calls
`resolve(strings.TrimPrefix(string(node.Destination), "alaws:"))` and, if
`ok`, replaces `node.Destination` with `[]byte(href)` before rendering
proceeds. This runs once per fragment render, is O(links in that fragment),
and requires no changes to goldmark's HTML writer itself.

Every call site supplies a different `resolve` closure, built once per
render from the already-in-scope `model.Lawbook`:

| Caller | `resolve(token)` returns |
|---|---|
| Static HTML (`html.Render`/`RenderAll`) | `"#" + anchorFor(resolved)` — see §4.2 for `anchorFor` |
| Static Markdown (`internal/renderer/markdown/markdown.go`) | `"#" + anchorFor(resolved)` (raw HTML anchor targets, see §4.4) |
| Static PDF (`internal/renderer/pdf/pdf.go`) | `"#" + anchorFor(resolved)` (internal PDF link, see §4.3) |
| Web UI (`pkg/alaws/render.go`'s `RenderedSections`) | an app hash route string, see §4.5 |

Where `anchorFor(resolved Resolved) string` is:

```go
func anchorFor(r Resolved) string {
	switch r.Kind {
	case KindLaw:
		if r.Law.Slug != "" {
			return r.Law.SectionID + "." + r.Law.Slug
		}
		return r.Law.Number // fallback for a law that has no slug yet
	case KindSection:
		return r.Section.ID
	}
	panic("unreachable")
}
```

Note this makes the anchor `id=` and the `href=` that points at it always
byte-identical (`sectionID.slug`, or `Number`/`ID` as fallback) — no separate
id-prefix bookkeeping needed beyond what already exists for combined
multi-book exports (see next paragraph).

### 4.2 Static HTML (`internal/renderer/html/html.go`)

Today, section headings get `id=%q` from `idPrefix+s.ID` and laws get
`id=%q` from `idPrefix+law.Number` (html.go:104-136, confirmed in
`renderSections`). `idPrefix` is `""` for a single-book export and
`"book%d-"` for the combined multi-book export (`RenderAll`, html.go:90-91),
to avoid collisions across independently-authored books that might reuse the
same section id.

Change the law `id=` to use `anchorFor` (§4.1) instead of `law.Number`
directly:

```go
anchor := law.Number
if law.Slug != "" {
	anchor = law.SectionID + "." + law.Slug
}
fmt.Fprintf(w, "<li id=%q>...", html.EscapeString(idPrefix+anchor))
```

Wire the `resolve` closure for both `Render` and `RenderAll` to call
`resolver.Resolve(book, token)` (or, for `RenderAll`, resolve against
whichever of the combined books' `model.Lawbook`s contains a match — loop
over all of them) and return `"#" + idPrefix + anchorFor(resolved)` on
success.

### 4.3 Static PDF (`internal/renderer/pdf/pdf.go`)

Mechanism confirmed by prior investigation: PDF generation is Markdown → PDF
via `github.com/stephenafamo/goldmark-pdf` (a goldmark `Renderer`
implementation built on `go-pdf/fpdf`), wired in `pdf.go:1-19,33-37`
(`markdownPDF = goldmark.New(goldmark.WithExtensions(extension.GFM),
goldmark.WithRenderer(pdf.New(pdf.WithEscapeHTML(false))))`).
`buildMarkdownInto` (pdf.go:62-118) converts the Lawbook IR into a Markdown
string fed through that pipeline; laws render as a bold-prefixed paragraph
(pdf.go:83-85: `fmt.Fprintf(b, "**%s** %s\n\n", law.Number, law.Text)`), not
as a heading.

The vendored library already supports internal PDF links end to end on the
*link source* side: its `WriteLink` special-cases any href starting with `#`
into `Pdf.WriteInternalLink(...)` instead of an external URL link. So once
`alaws:` tokens are rewritten to `#anchor` hrefs (same `resolve` plumbing as
everywhere else), clicking a link in the exported PDF already works — with
zero new PDF-library work — **provided an anchor target was registered for
that `anchor` string**.

Anchor *target* registration is the missing half. Today it only happens for
headings: `renderHeading` in the library reads a heading's AST `id`
attribute and calls `w.Pdf.AddInternalLink(string(anchor))`. Laws aren't
headings, so nothing registers an anchor for them. Implement this:

1. In `buildMarkdownInto`, immediately before each law's paragraph line,
   when that law has a computed `anchorFor(...)` value, emit a raw HTML
   comment sentinel: `fmt.Fprintf(b, "<!--alaws-anchor:%s-->\n", anchor)`.
   (Raw HTML nodes pass through goldmark's default parser untouched; this
   is the same "inert marker" trick used for the Markdown-export raw anchor
   in §4.4, just consumed differently here.)
2. Register a custom node renderer for `ast.KindRawHTML` on the
   `markdownPDF` goldmark-pdf renderer via the library's `pdf.WithNodeRenderers(...)`
   option (already exposed — confirmed present in the vendored library's
   `option.go`). The custom renderer: if the raw HTML content matches
   `^<!--alaws-anchor:(.+)-->$`, call `w.Pdf.AddInternalLink(matched-group)`
   and render zero visible output (don't fall through to the library's
   default raw-HTML handling, which would otherwise print nothing useful
   anyway since `WithEscapeHTML(false)` + no HTML tag support in a PDF
   context — but be explicit and intentional about the no-op rather than
   relying on that).
3. This is the exact same mechanism (`AddInternalLink` off an `id`-bearing
   AST node) the library already uses for headings, just triggered from a
   different node kind — wiring up an existing, currently-unused library
   capability, not new PDF plumbing, and not a fork of the dependency.

Wire the `resolve` closure for `pdf.Render`/`RenderAll` identically to
html.go's (§4.2): `resolver.Resolve` then `"#" + anchorFor(resolved)`
(no `idPrefix` concept needed here unless/until combined multi-book PDF
export also needs disambiguation — mirror html.go's `idPrefix` handling if
so).

### 4.4 Static Markdown (`internal/renderer/markdown/markdown.go`)

Today this emits zero anchor syntax at all — confirmed:
`fmt.Fprintf(w, "**%s** %s\n\n", law.Number, law.Text)`, no `id`, no anchor,
nothing to link to (and section headings get no `{#id}` attribute either,
just a plain `## Number Title` line with the section's `ID` shown separately
as an inline code span below it, per the existing sample output).

Add a raw HTML anchor immediately before any law or section that has a
resolvable anchor:

```go
if anchor := anchorFor(...); anchor != "" {
	fmt.Fprintf(w, "<a id=%q></a>\n", anchor)
}
fmt.Fprintf(w, "**%s** %s\n\n", law.Number, law.Text)
```

GFM (which this renderer already targets, matching goldmark's `extension.GFM`
used elsewhere) passes raw inline HTML through untouched, so `<a id="..."></a>`
is a real jump target when the file is viewed rendered (e.g. on GitHub) and
completely inert, harmless markup in any plain-text/non-rendering context.
Wire `resolve` the same way as §4.2/§4.3.

### 4.5 Web UI

**Server side** — `pkg/alaws/render.go`'s `RenderedSections` (which produces
the `CommentaryHTML`/`LawHTML` fragments served by `handleCompile` in
`internal/server/api.go:189-217`) runs with the full `model.Lawbook` already
in scope, so it resolves fully server-side, no client-side link-rewriting
needed at all. Its `resolve` closure returns an app hash route:

```go
func(token string) (string, bool) {
	r, err := resolver.Resolve(book, token)
	if err != nil {
		return "", false
	}
	switch r.Kind {
	case KindLaw:
		lawAnchor := r.Law.Slug
		if lawAnchor == "" {
			lawAnchor = fmt.Sprintf("%d", r.Law.Index)
		}
		return fmt.Sprintf("#/books/%s/%s/%s", bookPath, r.Law.SectionID, lawAnchor), true
	case KindSection:
		return fmt.Sprintf("#/books/%s/%s", bookPath, r.Section.ID), true
	}
	return "", false
}
```

**Client side** — three changes, all additive to the existing hand-rolled
hash router (deliberately dependency-free per its own comment: "no
dependency, on purpose (this app has three routes; a router library would be
more code than it saves)" — keep that philosophy, do not introduce a router
library):

1. `web/src/router.ts` — the `Route` union's `{ name: "book"; path: string;
   section?: string }` variant (router.ts:5-8) gains an optional `law?:
   string` field. Update `parseHash` to read a third `/`-separated hash
   segment into it, and `navigate`/whatever builds the hash string to emit
   it when present.
2. `web/src/views/BookDetail.tsx` — each law `<li>` currently has no DOM
   `id` at all (only a Preact reconciliation `key={law.Number}` at
   BookDetail.tsx:505, which never reaches the DOM). Add a real `id`:
   `id={`law-${law.Slug || String(law.Index)}`}` (prefixed however avoids
   collision with other DOM ids on the page — a `law-` prefix is enough
   since this element only exists while its section is the one currently
   selected/rendered).
3. Add a `useEffect` keyed on the route's `law` field that finds
   `document.getElementById('law-' + route.law)` and calls
   `scrollIntoView({behavior:"smooth", block:"center"})` on it — this is a
   near-verbatim copy of the existing search-highlight-scroll effect
   already in the file at BookDetail.tsx:116-124 (`useEffect` +
   `requestAnimationFrame` + `scrollIntoView`), just retargeted from
   `.search-highlight` to the new law-id lookup.

No click-interception is needed: the rendered `<a href="#/books/...">` links
are real hrefs, and the existing hash router already listens for
`hashchange` (confirmed: `location.hash` is only read/written inside
`router.ts` today) — clicking one just works once (1) and (2) above exist to
give it something to land on.

### 4.6 JSON export

No renderer change needed. `pkg/alaws/render.go`'s JSON path
(`json.NewEncoder(w).Encode(b.lawbook)`) already dumps the full
`model.Lawbook` — once `Law.Slug` exists on the model (§6, Phase 1), it's
in the JSON automatically.

---

## 5. Validator diagnostics

Add to `internal/validator/validator.go`, following the existing
`Diagnostic{Severity, Code, ...}` shape (validator.go:37; existing examples
at validator.go:87-125, e.g. `duplicate-id` at Error severity,
`missing-laws` at Warning severity):

| Code | Severity | Condition |
|---|---|---|
| `missing-slug` | Warning | A law has no `Slug`. (Required-by-policy per the user's decision, but non-fatal by default — see §6 "backfill" for how the shipped corpus reaches full compliance instead of just being flagged.) |
| `invalid-slug` | Error | `Slug` is non-empty but doesn't match `^[a-z][a-z0-9-]*$`. |
| `duplicate-slug` | Error | Two or more laws **within the same section** share a `Slug`. (Scope is per-section, not lawbook-wide — see §2.) |
| `ambiguous-identity` | Warning | A law's fully-qualified identity (`SectionID + "." + Slug`) is byte-identical to some *other* section's `ID` elsewhere in the book. Per the `Resolve` precedence in §2.1, the section-id match always wins in this case, silently shadowing the law — this diagnostic surfaces that instead of leaving it silent. |
| `dangling-reference` | Warning | An `alaws:` link anywhere in any section's commentary or law text fails to resolve via `Resolve` (§2.1). Implement as a post-compile pass: walk every section's raw `Commentary` and every law's `Text` for `alaws:` link destinations (reuse the same goldmark AST-walk approach as §4.1, or a simpler regex scan of Markdown link destinations `\(alaws:([^)]+)\)` if that's simpler to wire into the validator, which doesn't otherwise use goldmark) and call `Resolve` on each token. |

---

## 6. Implementation task list

### Phase 1 — model, parsing, identity, tooling

1. **`internal/model/model.go`** — add `Slug string` to `Law` (model.go:19-25).
   Update its doc comment to state: `Slug`, when present, is the law's
   stable identity (combined with its section's `ID` — see docs/linking.md
   §2); falls back to `SectionID+Index` only when absent.

2. **`internal/parser/parser.go`** — add `Slug string` to `RawLaw`
   (parser.go:46-50). Add the `slugAttrRe` regex from §3.1. Apply it at both
   `current.Text = strings.TrimSpace(current.Text)` sites in `parseLawLines`
   (parser.go:185 and parser.go:200): after trimming, check `slugAttrRe`
   against the end of `current.Text`; if it matches, set
   `current.Slug = match[1]` and re-set `current.Text` to the string with
   the match removed (then re-trim).

3. **`internal/numbering/numbering.go`** (`Assign`) — carry `RawLaw.Slug`
   through unchanged into the `model.Law.Slug` it constructs alongside
   `Number`/`Index`/`SectionID` (numbering.go:46-52 is where `Laws[j].Number`
   etc. are currently assigned — add `Laws[j].Slug = rawLaw.Slug` there,
   or wherever the `model.Law` is actually constructed in this package;
   confirm the exact construction site by reading the file, since the line
   numbers above are from the earlier research pass and the exact
   assignment site should be re-verified against current source before
   editing).

4. **`internal/resolver/resolver.go`** — implement `Resolve` exactly as
   specified in §2.1. Keep `ResolveLaw`/`ResolveSection` working as they do
   today (existing callers depend on their current signatures/behavior);
   extend `ResolveLaw`'s existing loop to also match `l.Slug == citation`
   (today it only checks `l.Number == citation`).

5. **`internal/validator/validator.go`** — implement all five diagnostics
   from §5.

6. **`internal/lawedit/lawedit.go`** — this is a **necessary fix, not scope
   creep**: `splitAtLawsMarker` (lawedit.go:21-49) reimplements clause
   parsing with its own bare `lawLineRe` and a fold loop that has *no fence
   awareness at all* — it joins every continuation line (fence content
   included) with a space, unconditionally. This means today, right now,
   before this feature exists, running `alaws law add` or `alaws law
   remove` on any section file that contains a multi-line or fenced law
   silently collapses it onto one line and destroys its formatting. It
   would do the same to a standalone-line `{#slug}` the moment this feature
   ships, unless fixed. Fix: replace `splitAtLawsMarker`'s hand-rolled loop
   with a call into `parser.parseLawLines` (parser.go, already
   fence-aware — this may require exporting it or a small wrapper, since
   it's currently unexported; check and adjust visibility as needed), so
   `Add`/`Remove` round-trip clauses (and their slugs, and their fenced code
   blocks) losslessly. `writeClauses` (lawedit.go:51-63) must then also
   serialize each clause's slug back out — inline (`N. text {#slug}`) if
   the clause has no embedded newlines, or on its own trailing line if it
   does.
   - Add `SetSlug(path string, citation string, slug string) error` (new
     function in this package) for the CLI command in task 7. `citation`
     here should accept anything `resolver.Resolve` accepts (number or
     existing slug) to locate the target clause before rewriting it.
   - `Add` gains an auto-slug path: when the caller doesn't supply an
     explicit slug, derive one from the law's text (lowercase, strip
     punctuation, take the first ~5-6 significant words, join with `-`),
     de-duplicated against the other slugs already present *in that
     section* (per-section uniqueness, §2) by appending `-2`, `-3`, etc. on
     collision. This is what makes "slugs required on every law" (§5,
     `missing-slug`) frictionless going forward: laws added through the CLI
     are never slug-less.

7. **`internal/cli`** (package location: confirm exact file under
   `internal/cli/`, likely alongside the existing `compile.go`'s `show`
   command and wherever `alaws law add`/`alaws law remove` are already
   wired to `internal/lawedit`) — add:
   - `alaws law slug <book> <citation-or-slug> <new-slug>` → calls
     `lawedit.SetSlug`. Validate `<new-slug>` against the charset (§3.1) and
     re-run the validator's `duplicate-slug`/`invalid-slug` checks (scoped
     to the target law's section) *before* writing, so a bad rename fails
     with a clear CLI error instead of writing invalid Markdown.
   - `alaws law fill-slugs <book>` (or a `--fill-missing` flag on an
     existing command — pick whichever fits the existing CLI command
     conventions better once you're looking at `internal/cli`) → sweeps
     every law lacking a slug in the given book and assigns one via the
     same auto-generation logic as `Add`'s new auto-slug path (factor that
     logic into a shared internal function both call, don't duplicate it).
   - Also update the CLI's `show` command (`internal/cli/compile.go:123-152`)
     to call the new `resolver.Resolve` instead of its current manual
     try-`Resolve`-then-`Section` fallback — same behavior, less duplicated
     logic.

8. **Backfill the shipped corpus** — run the new `fill-slugs` tool against
   `fixtures/basic`, every lawbook under `examples/` (there are three:
   `examples/engineering`, `examples/payments`, `examples/support`, plus
   `examples/integration` — confirm the exact set by listing `examples/`),
   and regenerate everything under `samples/` (these are pre-built exports
   checked into the repo — regenerate via whatever `make` target already
   produces them, e.g. `make build && ./alaws export ...` matching the
   existing `samples/` generation process; check `Makefile` for the exact
   existing command). This is required so `make test`/`make build` stay
   green with the new `missing-slug` diagnostic active, and so the shipped
   examples actually demonstrate the feature working, not just document it.

### Phase 2 — rendering and cross-format links

Implement exactly as specified in §4.1 through §4.6 above, in this order
(each step is independently testable before moving to the next):

1. §4.1 — `RenderFragment`'s new `resolve` parameter and the goldmark
   `alaws:` link-rewriting transform. Update every existing call site to
   pass a resolver (even a stub one initially) so the build stays green
   while the rest of Phase 2 lands incrementally.
2. §4.2 — static HTML anchors + resolver wiring.
3. §4.3 — static PDF anchor registration (`WithNodeRenderers` +
   `<!--alaws-anchor:...-->` sentinel) + resolver wiring. This is the
   riskiest single step (new integration with a third-party rendering
   library's extension points) — budget the most test time here (see §7).
4. §4.4 — static Markdown raw-HTML anchors + resolver wiring.
5. §4.5 — web UI: server-side resolver in `RenderedSections`, then
   `router.ts`'s `law` field, then `BookDetail.tsx`'s DOM `id` + scroll
   effect, in that order (each is a small, separately-verifiable diff).
6. §4.6 — confirm JSON export carries `Slug` with no code change needed
   (just a verification step, not an implementation step).

---

## 7. Test plan

- **`internal/parser`**: table-driven tests for `slugAttrRe` extraction —
  (a) inline-trailing on a one-line law, (b) own-line after a plain
  multi-line law, (c) own-line after a fenced code block, (d) negative case:
  a law whose real text ends in something brace-shaped but not matching the
  slug charset (e.g. uppercase, or starting with a digit) must be left
  completely untouched in `Text` with `Slug` empty.
- **`internal/validator`**: one test per new diagnostic code in §5,
  mirroring the existing table-driven style already in
  `validator_test.go` (see its `codes()` helper and the `unfenced-json`
  tests for the pattern to follow).
- **`internal/resolver`**: tests for every branch of the `Resolve`
  precedence order in §2.1 — exact section-id match, fully-qualified
  `section.slug` match, citation-number match, section-number match,
  unambiguous bare-slug match, ambiguous bare-slug (must error with the
  specific ambiguity message), and total non-match (must error via
  `ErrNotFound`).
- **`internal/lawedit`**: a regression test — construct a fixture section
  file with a fenced, multi-line law that has an own-line `{#slug}`, run
  `Add` (of an unrelated new law) then `Remove` (of that unrelated law),
  and assert the original fenced law's `Text` *and* `Slug` are byte-for-byte
  unchanged after the round trip. This is the pre-existing bug described in
  task 6 of §6, being fixed and regression-tested together, not just the
  new feature.
- **`internal/renderer/pdf`**: a test that compiles a fixture with one
  slugged law referenced by an `alaws:` link from another section, renders
  to PDF, and confirms (via whatever inspection the `go-pdf/fpdf`/
  `goldmark-pdf` stack allows — check if the vendored library exposes a way
  to enumerate registered internal links/annotations for test purposes, or
  fall back to a byte-level sanity check that the output PDF contains a
  `/Link` annotation object referencing the expected name) that an internal
  link annotation was actually produced. Budget extra time here since this
  is new integration with third-party rendering internals — see §4.3's
  "riskiest step" note.
- **End to end**: `make test` after the Phase 1 backfill (§6 task 8) — the
  full existing fixture/example/sample corpus must compile clean under the
  new `missing-slug` diagnostic.
- **Manual browser check** (per `AGENTS.md`'s own existing rule, "don't test
  the web UI with Go tests — use the browser for that"): `make serve`, open
  a book, click an `alaws:` link inside a law's rendered commentary text,
  confirm the page navigates to and scrolls/highlights the correct law.
  Then run `alaws compile <book> --format html,pdf,md` on the same book and
  confirm: the HTML export's in-page link jumps to the right anchor; the PDF
  export's link (open the PDF in a viewer with internal-link support, e.g.
  any modern browser's built-in PDF viewer, or Preview on macOS) jumps to
  the right page/law; the Markdown export's raw `<a id>` anchors are present
  immediately before the referenced law when viewed rendered (e.g. paste
  into a GitHub-rendered view or any GFM-rendering preview).

---

## 8. Open items to flag back to the user during/after implementation

- The `missing-slug` diagnostic defaults to Warning severity even though the
  user's decision was "required on every law." This was a deliberate
  engineering call (documented in §5) to avoid a hard compile failure for
  any hand-edited lawbook that hasn't been through the backfill tool yet;
  full compliance for the *shipped* corpus is achieved by actually running
  the backfill (§6 task 8), not by relaxing the rule. If the user wants
  `missing-slug` to be a hard `Error` instead, that's a one-line severity
  change in §5's table — flag this back to them once Phase 1 is otherwise
  done, rather than deciding unilaterally.
- §2's interpretation of the user's shorthand example
  `[law-name](a.b.c.law-name)` — this document keeps the `alaws:` scheme
  prefix (from the earlier locked decision "Standard MD link + alaws:
  scheme") and treats `a.b.c.law-name` as the *token after the colon*, not
  as a bare href with no scheme at all. If that's not what was meant, the
  fix is confined to §3.2's syntax and the `alaws:` prefix-stripping in
  §4.1 — nothing else in this document depends on which choice is made.
