# AgentLaws Implementation

This document describes the intended implementation architecture and v1 design for AgentLaws (`alaws`).

It is deliberately more technical than the README. The README explains what AgentLaws is; this document explains how the system should be built and which design decisions should constrain the implementation.

The guiding principle is:

> **Build a deterministic lawbook compiler and provenance system first. Add richer governance semantics later.**

---

# 1. Architecture Overview

AgentLaws has four primary layers:

```text
                         AgentLaws

┌──────────────────────────────────────────────────────────┐
│                      Source Lawbooks                     │
│                                                          │
│  alaws.toml + Markdown files                            │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                     Parser / Validator                   │
│                                                          │
│  metadata / commentary / laws / ordering / references   │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                       Lawbook IR                         │
│                                                          │
│  sections / clauses / numbering / identities / source  │
└───────────────┬──────────────────────┬───────────────────┘
                │                      │
                ▼                      ▼
      Human compilation          Runtime extraction
       HTML / PDF / UI            agent law context
                │                      │
                └──────────┬───────────┘
                           ▼
                  Provenance / History
                           │
                           ▼
                         Git
```

The same Lawbook IR should drive:

* CLI operations
* HTML generation
* PDF generation
* live compilation
* the local web UI
* Go library calls
* agent-facing law extraction
* citation resolution
* provenance lookup

There should be one canonical interpretation of the source.

---

# 2. Core Design Constraints

The implementation should preserve the following invariants.

## 2.1 Folders have no semantic meaning

Directory layout is purely for human organization.

These are semantically equivalent:

```text
security/secrets.md
```

and:

```text
foo/bar/secrets.md
```

provided the file is referenced identically in the lawbook ordering and has equivalent metadata/content.

Moving a file between directories must not silently change its place in the lawbook.

---

## 2.2 `alaws.toml` ordering is authoritative

Every participating Markdown file must explicitly occur in the ordering.

For example:

```toml
ordering = [
  "principles.md",
  "security/authentication.md",
  "security/secrets.md",
  "coding.md",
]
```

Do not recursively discover and implicitly include files.

File discovery is useful for diagnostics, but not for determining the lawbook.

This enables warnings such as:

```text
warning: security/legacy.md exists but is not present in ordering
```

---

## 2.3 Source identity is different from presentation numbering

Every section has a stable source ID:

```yaml
id: engineering.security
```

A clause receives a canonical law number during compilation:

```text
2.5.3
```

These have different purposes.

```text
engineering.security
    → stable identity

2.5.3
    → citation into a particular compiled lawbook
```

Do not use presentation numbers as persistent IDs.

---

## 2.4 Commentary and laws are structurally distinct

Every section contains:

```text
Metadata
Commentary
Laws
```

The structural delimiters should be HTML comments:

```md
<!-- alaws:commentary -->

...

<!-- alaws:laws -->

1. ...
2. ...
3. ...
```

Do not use:

```md
## Commentary
## Laws
```

as structural markers.

The former would unnecessarily make the structural distinction part of the rendered document.

The parser should recognize only the reserved AgentLaws markers.

---

## 2.5 Both commentary and laws are normal compiled content

HTML and PDF contain both:

```text
Commentary
Laws
```

There is no separate "Agent Text" artifact.

Agent-facing content is simply an extraction of selected law clauses from the same Lawbook IR.

---

# 3. Repository and Lawbook Layout

A repository can contain multiple independent lawbook clusters.

Example:

```text
payments/
  alaws.toml
  authorization.md
  refunds.md

support/
  alaws.toml
  customer-data.md
  escalation.md
```

Each `alaws.toml` defines one cluster.

Clusters may represent:

* a module
* a subsystem
* a team
* an agent
* a domain
* any other independently governed collection

AgentLaws should not force a particular organizational model.

---

# 4. `alaws.toml`

The initial configuration should remain deliberately small.

Conceptually:

```toml
title = "Engineering Governance"

ordering = [
  "principles.md",
  "security/authentication.md",
  "security/secrets.md",
  "coding.md",
  "operations.md",
]
```

Potential configuration areas can be added later, including explicit lawbook storage, but the initial implementation should avoid configuration proliferation.

The important field is:

```toml
ordering = [...]
```

This is the authoritative sequence.

---

# 5. Repository-Local and Global Storage

AgentLaws needs a way to manage lawbooks independently of the current project.

The intended model is:

```text
~/.alaws/
```

for global storage.

A repository can establish a repository-local AgentLaws root with:

```text
.alaws/
```

For example:

```text
repository/
├── .alaws/
│   ├── ...
├── src/
└── ...
```

The exact storage mechanism is expected to support explicit configuration as well.

A simple resolution hierarchy is:

```text
explicit configuration
        >
repository-local .alaws/
        >
user-level ~/.alaws/
```

The implementation should keep this mechanism simple and deterministic.

---

# 6. Section File Format

A section is Markdown with YAML frontmatter and two structural regions.

Example:

```md
---
title: Security
id: engineering.security
---

<!-- alaws:commentary -->

This section defines security requirements for agents
working with this repository.

The commentary contains rationale, discussion, history,
trade-offs, examples, and other information intended
primarily for human readers.

<!-- alaws:laws -->

1. Credentials must never be committed to source control.

2. Agents must not print credentials into logs.

3. Credentials discovered in source must be treated as compromised.
```

Required metadata:

```yaml
title: ...
id: ...
```

Optional metadata:

```yaml
level: 2
```

The initial parser should reject malformed or missing required metadata.

---

# 7. Markdown Handling

The Markdown content should remain ordinary Markdown as much as possible.

Arbitrary Markdown should be usable inside commentary and laws.

The two distinctions AgentLaws needs to understand are:

```text
commentary region
laws region
```

Within those regions, Markdown is primarily presentation/source content.

The compiler should not initially attempt sophisticated semantic interpretation of prose.

---

# 8. Heading Levels

The section file's `title` becomes the title/heading associated with that source file.

By default, its heading level should be derived from the compiled ordering structure rather than filesystem depth.

Important:

> **Filesystem depth must never determine heading level.**

The filesystem has no semantic meaning.

The implementation needs a defined rule for the default level based on the lawbook's ordered structure. Where the default is inappropriate, metadata can override it:

```yaml
level: 2
```

This should only override presentation hierarchy; it does not alter the lawbook ordering itself.

Markdown headings inside a section remain normal Markdown content and can be used by the author freely.

---

# 9. Law Representation

The v1 definition of a law is intentionally simple.

A law is a numbered list item inside:

```text
<!-- alaws:laws -->
```

For example:

```md
1. Credentials must never be committed to source control.

2. API keys must be stored using the approved secret store.

3. Credentials discovered in source must be treated as compromised.
```

Do not attempt to build a formal policy language in v1.

AgentLaws does not initially need to understand:

* exceptions
* permissions
* prohibitions
* conditions
* subjects
* environments
* operations
* precedence
* conflict semantics

Those concepts may appear in prose, but the compiler does not need to model them.

---

# 10. Law Numbering

The author is encouraged to number every clause.

The numbering should remain at the clause level rather than requiring authors to maintain large globally unique identifiers.

The lawbook compiler assigns the canonical section number and combines it with the clause number.

Conceptually:

```text
section:
2.5

source clauses:

1. ...
2. ...
3. ...

compiled:

2.5.1 ...
2.5.2 ...
2.5.3 ...
```

The compiler owns canonical numbering.

Authors should never need to manually update `2.5.3` after moving a section.

---

# 11. Numbering Validation

The compiler should inspect the laws region and emit diagnostics.

At minimum:

### Error

* laws region missing
* malformed Markdown structure preventing identification of laws
* invalid metadata

### Warning

* laws region contains substantial prose but no numbered list
* numbered list appears malformed
* numbered list is empty
* numbering is unexpectedly sparse
* non-list content appears between clauses where it may indicate an authoring mistake

The goal is to encourage authors toward a predictable machine-addressable structure without requiring a formal language.

---

# 12. Lawbook Intermediate Representation

The central internal representation should be independent of Markdown, HTML, PDF, or CLI syntax.

Conceptually:

```go
type Lawbook struct {
    Metadata   LawbookMetadata
    Sections   []Section
    Provenance Provenance
}

type Section struct {
    ID          string
    Title       string
    Level       int
    Source      SourceRef
    Commentary  Markdown
    Laws        []Law
}

type Law struct {
    Number      string
    Text        Markdown
    SectionID   string
    Source      SourceRef
}
```

The exact types can evolve.

The important architectural rule is:

> Renderers and consumers operate on the IR, not directly on Markdown.

---

# 13. Source Provenance

Every section and law should retain enough source information to locate its origin.

At minimum:

```go
type SourceRef struct {
    Path       string
    LineStart  int
    LineEnd    int
}
```

A law should therefore be resolvable from:

```text
2.5.3
```

to:

```text
section ID
source file
source lines
exact clause
```

This is the basis for history, diagnostics, editor integration, and auditability.

---

# 14. Stable Identity Model

The source document ID is the stable identity.

Example:

```yaml
id: engineering.security
```

A law can internally be represented by:

```text
engineering.security + clause index
```

The external citation remains:

```text
2.5.3
```

The internal mapping might be:

```text
2.5.3
   ↓
engineering.security
   ↓
security.md
   ↓
clause 3
```

This makes it possible to change presentation numbering without losing the identity of the source section.

---

# 15. Citation Resolution

Citation lookup should be a core library capability.

Conceptually:

```go
law, err := book.Resolve("2.5.3")
```

The result should expose:

```text
number
text
section ID
section title
source path
source location
lawbook revision
history information
```

A higher-level history API may then support:

```go
book.History("2.5.3")
```

or:

```go
book.Resolve("2.5.3").History()
```

The exact public API can be finalized once the IR stabilizes.

---

# 16. Agent-Facing Extraction

Agent extraction is a consumer of the Lawbook IR.

For example:

```go
laws, err := book.Laws(...)
```

should return a representation containing the selected laws with canonical citation numbers.

Conceptually:

```text
2.5.1 Credentials must never be committed to source control.
2.5.2 Agents must not print credentials into logs.
2.5.3 Credentials discovered in source must be treated as compromised.
```

AgentLaws does not need to know how the resulting text is incorporated into the larger application prompt.

The application owns prompt assembly.

AgentLaws owns:

```text
which laws
their numbers
their source
their provenance
their rendering
```

---

# 17. Agent Citation Requirement

When a lawbook is supplied to an agent, applications should be able to instruct the agent to cite the applicable laws.

Example:

```text
Decision: Reject

Laws:
2.5.1
2.5.3
```

The important goal is that the model produces precise citations instead of relying solely on prose explanations.

AgentLaws should make citation parsing straightforward.

Potentially:

```go
citations := alaws.ExtractCitations(agentResponse)
```

would produce:

```text
2.5.1
2.5.3
```

This can later be tied into decision auditing.

---

# 17a. Variable Substitution in Law Text

Applications frequently need to insert dynamic values into law or commentary text when
composing a prompt for a specific API call — an agent name, a repository, a ticket ID, a date,
an environment. AgentLaws supports this without weakening the two invariants that matter most
for governance: deterministic compilation (§47) and tamper-evident signing (§49).

## Syntax

A placeholder is `{{identifier}}`, where `identifier` matches:

```text
[a-zA-Z_][a-zA-Z0-9_.]*
```

Dots are allowed for light namespacing, e.g. `{{env.region}}`. There are deliberately **no**
pipes, filters, conditionals, loops, or function calls — a law is not a template program. This
mirrors the reasoning in §60 for not executing MDX: governance text should stay data, not code.

A literal `{{` is written as `\{{`.

Placeholders may appear in law clause text and in commentary. They may **not** appear in
frontmatter metadata (`id`, `title`, `level`) — those values define the section's stable
identity and must remain readable without a render step.

## Compile-time vs. render-time

The compiler validates placeholder **syntax only** — balanced braces and a valid identifier —
and reports a new diagnostic code, `invalid-template`, for malformed placeholders (see §19). It
never resolves a placeholder to a value. The canonical Lawbook IR stores law and commentary
text with placeholders intact, exactly as written. This is what gets hashed and signed, so
compilation and provenance never depend on runtime variable values — two compilations of the
same source produce the same signed artifact regardless of what an application later renders it
with.

Resolution happens only at the extraction/render boundary, as a pure function over strings:

```go
type MissingPolicy int

const (
    MissingError           MissingPolicy = iota // fail if a placeholder has no value (default)
    MissingKeepPlaceholder                       // leave `{{x}}` untouched
    MissingEmpty                                 // substitute ""
)

func Render(text string, vars map[string]string, policy MissingPolicy) (string, error)
```

## Library surface

```go
laws, err := book.Laws(selector...)
rendered, err := laws.Render(alaws.RenderOptions{
    Vars: map[string]string{
        "agent_name": "ci-bot",
        "repo":       "org/app",
    },
    OnMissing: alaws.MissingError, // default
})
```

`Render` never mutates the Lawbook IR. It produces a new string suitable for inclusion in an
application's prompt. The default missing-variable policy is `MissingError`: an agent-facing
prompt should never silently ship an unresolved `{{...}}` into a live API call.

## CLI variable sources

The CLI accepts variables from two explicit sources, highest precedence first:

```text
--var key=value      (repeatable)
--vars-file f.json | f.yaml   (flat string map)
```

There is no implicit environment-variable pickup in v1. Inputs stay explicit so that scripted
or agent-driven CLI use remains predictable and auditable — the exact values used to render a
prompt should be visible in the command that produced it.

See `alaws render` in §32 for the CLI form.

---

# 18. Compilation Pipeline

`alaws compile` should conceptually execute:

```text
discover clusters
        ↓
load alaws.toml
        ↓
validate ordering
        ↓
load source files
        ↓
parse metadata
        ↓
split commentary / laws
        ↓
parse numbered laws
        ↓
construct Lawbook IR
        ↓
assign presentation hierarchy
        ↓
assign canonical law numbers
        ↓
run diagnostics
        ↓
generate artifacts
        ↓
generate provenance
        ↓
sign compilation
```

The implementation should keep these stages explicit.

Avoid putting parsing, validation, numbering, rendering, and signing into one giant compilation function.

---

# 19. Diagnostics

Diagnostics should be structured internally even if initially rendered as CLI text.

Conceptually:

```go
type Diagnostic struct {
    Severity Severity
    Code     string
    Message  string
    Source   *SourceRef
}
```

Possible codes:

```text
missing-config
missing-file
unused-file
missing-title
missing-id
duplicate-id
missing-commentary
missing-laws
invalid-laws
invalid-ordering
invalid-metadata
invalid-template
```

`invalid-template` covers malformed `{{...}}` placeholders in law or commentary text — see §17a
for the variable substitution model. It is a syntax check performed at compile time; it does not
mean a variable is missing a value, since values are only resolved at render time.

A structured diagnostic model will make it easier for the future web UI to display the same errors as the CLI.

---

# 20. Errors vs Warnings

Compilation should distinguish problems that invalidate a lawbook from problems that merely deserve attention.

Example:

```text
ERROR:
security/secrets.md is listed in ordering but does not exist

WARNING:
security/legacy.md exists but is not listed in ordering
```

The general rule should be:

> If PromptGov cannot deterministically understand the lawbook, compilation fails.

Warnings should be reserved for situations where the lawbook remains deterministic but the source probably contains an unintended omission or quality problem.

---

# 21. Detecting Unordered Files

The compiler should scan the cluster directory for Markdown files.

That scan is diagnostic only.

For every Markdown file not referenced by `ordering`, report a warning.

For every path referenced by `ordering` that does not exist, report an error.

This creates a useful invariant:

```text
ordered files = included files
unlisted files = excluded files
```

There is no implicit recursive inclusion.

---

# 22. HTML Rendering

HTML should be generated from the Lawbook IR.

The HTML should contain:

```text
title
sections
commentary
laws
```

Canonical law numbers should be visible.

The HTML may also include provenance information, such as:

```text
Lawbook revision
Compilation timestamp
PromptGov version
Git revision
Compiler identity
```

The implementation should keep provenance presentation separate from provenance data.

---

# 23. PDF Rendering

PDF should use the same Lawbook IR and should not have a separate semantic rendering pipeline.

The desired architecture is:

```text
Lawbook IR
   ├── HTML renderer
   └── PDF renderer
```

rather than:

```text
Markdown
   ├── HTML parser
   └── PDF parser
```

This ensures that HTML and PDF have identical governance semantics.

---

# 24. Provenance Manifest

Compilation should produce a canonical manifest containing information about the compiled state.

Conceptually:

```json
{
  "lawbook": "engineering",
  "revision": "8f3a91c",
  "compiled_at": "2026-08-18T...",
  "promptgov_version": "...",
  "compiler": {
    "name": "...",
    "email": "..."
  },
  "signature": "...",
  "sections": [
    {
      "id": "engineering.security",
      "source": "security/secrets.md",
      "clauses": 4
    }
  ]
}
```

The exact schema is not final.

The essential requirement is that the manifest identifies:

```text
what was compiled
who compiled it
from what Git state
when
with what AgentLaws version
and how its authenticity can be checked
```

---

# 25. Signing Model

Compilation should support attribution and cryptographic signing.

These should be considered two distinct concepts.

## Git identity

AgentLaws can obtain:

```text
git config user.name
git config user.email
```

to determine the local developer identity.

## Cryptographic signature

Where a Git signing mechanism is configured, AgentLaws should be able to associate the compilation with a cryptographic identity.

The exact signing implementation should be chosen after examining the Git signing mechanisms available in the target environments.

The important invariant is:

> **The signature must cover the canonical lawbook representation, not renderer-specific HTML or PDF bytes.**

This allows multiple representations of the same governed state to share one provenance identity.

---

# 26. Compilation Artifacts

A compiled cluster should have a clear artifact model.

Conceptually:

```text
.alaws/
    build/
        manifest.json
        lawbook.json
        lawbook.html
        lawbook.pdf
```

The exact directory name can evolve.

The important artifacts are:

```text
canonical Lawbook representation
human HTML
human PDF
provenance manifest
```

There is no separate "agent text" artifact.

Agent context is obtained programmatically from the lawbook.

---

# 27. Live Compilation

`alaws watch` should provide a simple development loop:

```text
filesystem watcher
        ↓
source changed
        ↓
recompile
        ↓
regenerate HTML
        ↓
notify browser
```

Initially, correctness should be prioritized over sophisticated incremental compilation.

For the expected scale of a lawbook, recompiling the affected cluster on changes should be entirely acceptable.

The watcher can later become incremental if profiling demonstrates a need.

---

# 28. Local Web UI

The UI should be a local Preact application.

The Go process should provide the local API/server.

The UI should consume the same Lawbook IR and diagnostics used by the CLI.

The initial UI should focus on:

* lawbook navigation
* section viewing
* commentary/law presentation
* compiler diagnostics
* live refresh
* ordering management

## Visual style: strictly VS Code theming

The UI must look and feel like VS Code, not merely be "a Preact app." Concretely:

* All color, spacing, and font values come from VS Code's standard CSS custom-property names —
  `--vscode-editor-background`, `--vscode-foreground`, `--vscode-font-family`,
  `--vscode-focusBorder`, `--vscode-list-hoverBackground`, `--vscode-panel-border`, and so on.
  No colors are hardcoded outside that token layer.
* Because the app runs standalone in a normal browser rather than inside an actual VS Code
  webview, it ships its own default values for those custom properties — one set matching VS
  Code's Dark+ theme, one matching its Light+ theme — rather than relying on VS Code to inject
  them.
* Layout follows VS Code's visual language: a flat, minimal-chrome tree view for navigation
  (the lawbook's chapters/sections) on the left, a detail/reading pane on the right, monospace
  or VS Code's default UI font, no rounded cards, gradients, or shadow-heavy "modern SaaS"
  styling.

This is a constraint on the CSS layer of the `web/` app (see §65), not a new architectural
component.

---

# 29. Drag-and-Drop Ordering

The UI should allow users to reorder files visually.

Example:

```text
1. Principles
2. Security
3. Coding
4. Operations
```

Dragging:

```text
Operations
```

above:

```text
Coding
```

must result in a corresponding edit to:

```toml
ordering = [
  "principles.md",
  "security/secrets.md",
  "operations.md",
  "coding.md",
]
```

The UI should modify the source `alaws.toml`.

It should not create a database containing an alternative ordering.

Git must be able to show the change directly.

---

# 30. UI and Filesystem Boundaries

The web UI should treat source files as authoritative.

It should never silently create an internal representation that cannot be reproduced in the files.

The desired relationship is:

```text
Files
  ↕
Lawbook compiler
  ↕
Web UI
```

rather than:

```text
Files → UI database → generated files
```

The latter would make Git history and external editing substantially harder.

---

# 31. Watch Mode and UI

A useful development flow should be:

```bash
alaws watch
```

then open the local lawbook in a browser.

Source edits from:

* VS Code
* Vim
* another editor
* the AgentLaws UI

should all converge on the same source files.

The browser representation should update after successful compilation.

Diagnostics should be surfaced immediately when compilation fails.

---

# 32. CLI Design

The CLI is the primary interface agents use to work with a lawbook, so it needs to be complete
and predictable rather than minimal. It is organized around four resources that map directly
onto the Lawbook IR (§12), plus lawbook-level operations.

## Resource model

* **book** — a lawbook cluster: a directory containing `alaws.toml` (README's "lawbook" =
  this document's "cluster", §3).
* **chapter** — a top-level `Section` (`level: 1`, no parent), listed directly in `ordering`.
  A chapter typically holds commentary and may also contain its own laws.
* **section** — a `Section` at `level ≥ 2`, created under a specific parent chapter. A
  section's parent is *derived*, not a stored field: it is the nearest preceding `ordering`
  entry whose level is lower than its own — the same outline rule already implied by the
  heading-level model in §8. `section create --parent <chapter-id>` computes the correct
  insertion index (immediately after the parent's last existing descendant) and defaults
  `level` to `parent.level + 1` unless `--level` overrides it.
* **law** — a numbered clause inside a section's `<!-- alaws:laws -->` region.

Chapters and sections are not a new persisted concept — both are ordinary `Section` files.
"Chapter" vs "section" is CLI/library vocabulary for "top-level" vs "nested" sections, chosen
because it matches how people actually talk about a lawbook (README's "Lawbook Analogy").

All ordering mutations (`chapter`/`section` create, move, remove) go through one shared library
function that edits `alaws.toml` in place — the same function the drag-and-drop UI (§29) calls.
There is exactly one code path that writes ordering (§30, §52). Law-region mutations
(`law add`/`remove`) go through a separate, narrower function that locates the
`<!-- alaws:laws -->` marker and edits only its numbered list, leaving the rest of the file
untouched.

## Command reference

```text
alaws init [path] [--title "..."]                     Alias for `books create`

alaws books list [--root .] [--json]
alaws books create <path> --title "..."
alaws books show <path> [--json]

alaws chapter create <book> <file> --title "..." --id "..." [--after <id>|--position N]
alaws chapter list <book> [--json]
alaws chapter move <book> <id> [--before <id>|--after <id>|--position N]
alaws chapter remove <book> <id> [--force]

alaws section create <book> <file> --parent <chapter-id> --title "..." --id "..."
                      [--after <id>|--position N] [--level N]
alaws section list <book> [--chapter <id>] [--json]
alaws section show <book> <id> [--json]
alaws section move <book> <id> [--parent <chapter-id>] [--before|--after|--position]
alaws section remove <book> <id> [--force]

alaws law add <book> <section-id> "law text" [--after N]
alaws law list <book> <section-id> [--json]
alaws law remove <book> <section-id> <N> [--force]

alaws compile [book...] [--out dir] [--format html,json,pdf] [--strict]
alaws validate [book...] [--json]
alaws list [book] [--json]                             List compiled sections/laws
alaws show <citation-or-id> [--json]
alaws resolve <citation> [--json]
alaws history <citation> [--json]

alaws render --book <path> (--section <id> | --law <citation> | --all)
             [--var k=v]... [--vars-file f] [--on-missing error|keep|empty] [--json]

alaws watch [book] [--port 8420]
alaws serve [book] [--port 8420]                       Serve UI read-only, no watcher

alaws sign [book] [--key ...]
alaws verify [book] [--manifest path]
```

`alaws render` is the CLI entry point for the variable substitution model in §17a — it is how
an application or agent turns selected laws into prompt-ready, variable-resolved text from the
command line rather than the Go library.

## Cross-cutting behavior (applies to every subcommand)

* `--json` on every read command — a structured, machine-readable form of the same data the
  human-readable output shows, so agents can drive the CLI directly rather than scraping text.
* Exit codes: `0` success, `1` validation/compile error, `2` usage error, `3` not found
  (e.g. `resolve`/`show` given an unknown citation or ID).
* `--root <path>` — a global flag for locating a book when it isn't given explicitly, using the
  storage resolution hierarchy in §5.
* `--dry-run` on every mutating command — prints the change (new/edited files, the resulting
  `ordering` diff) without writing anything. Important for an agent that wants to preview a
  change before it is committed.
* Every command is a thin wrapper over the same `internal/`/`pkg/alaws` library calls used by
  the Go API and the UI — no command contains logic that doesn't also exist in the library
  (§52).

Do not overcommit to commands beyond this list before the core model (parser, compiler,
numbering) is stable — but the shape above is the intended v1 surface, not a sketch to be
redesigned per §64's milestones.

---

# 33. Library Packages

The Go library should be structured around reusable concepts rather than CLI commands.

A possible package organization:

```text
internal/
    parser/
    compiler/
    validator/
    numbering/
    renderer/
    provenance/
    signing/
    watcher/
    discovery/
    ordering/    # shared alaws.toml ordering mutation, used by CLI (§32) and UI (§29)
    lawedit/     # shared <!-- alaws:laws --> numbered-list mutation, used by `alaws law`
    template/    # variable substitution at render time (§17a)

pkg/
    alaws/
    model/
```

The exact public package structure can be simplified after the initial implementation proves useful.

The important boundary is that the CLI should consume the same library that third-party applications use.

---

# 34. Suggested Internal Modules

A more detailed internal decomposition:

```text
parser
    Parse TOML
    Parse frontmatter
    Parse Markdown
    Identify commentary/laws markers

discovery
    Find clusters
    Find Markdown files
    Detect unlisted files

validator
    Validate metadata
    Validate ordering
    Validate document structure
    Validate laws

model
    Lawbook
    Section
    Law
    SourceRef
    Provenance

compiler
    Construct Lawbook IR
    Assign hierarchy
    Assign law numbers
    Produce canonical representation

resolver
    Resolve law numbers
    Resolve section IDs
    Resolve source references

renderer/html
    Render human lawbook

renderer/pdf
    Render human lawbook

provenance
    Collect Git metadata
    Construct manifest

signing
    Sign canonical representation

watcher
    Monitor relevant files

ordering
    Read/write the `ordering` list in alaws.toml
    Compute chapter/section parent-child structure (§32)
    Compute insertion points for create/move operations
    Shared by the CLI (`alaws chapter`/`section`) and the UI's drag-and-drop editor (§29)

lawedit
    Locate the `<!-- alaws:laws -->` region in a section file
    Append/remove numbered clauses without disturbing surrounding Markdown
    Backs `alaws law add`/`alaws law remove`

template
    Validate `{{identifier}}` placeholder syntax at compile time (§17a)
    Resolve placeholders against a variable map at render time
    Backs `book.Laws(...).Render(...)` and `alaws render`

server
    Serve UI/API
```

---

# 35. Canonical Representation

A major implementation requirement is a deterministic representation of the compiled lawbook.

This representation should contain semantic content rather than renderer-specific details.

For example:

```json
{
  "sections": [
    {
      "id": "engineering.security",
      "title": "Security",
      "laws": [
        {
          "number": "2.5.1",
          "text": "Credentials must never be committed to source control."
        }
      ]
    }
  ]
}
```

The representation should be canonically serialized before hashing/signing.

This provides a stable input to provenance.

---

# 36. Content Hashing

Each significant object can eventually carry hashes.

Potential hierarchy:

```text
law text hash
        ↓
section hash
        ↓
lawbook hash
        ↓
signature
```

This is useful for determining whether a particular law actually changed even if the surrounding document or numbering changed.

The exact hashing scheme should be specified after the IR is stable.

---

# 37. Law History

Because law numbers are presentation-oriented, history lookup should ultimately operate through stable section identity plus clause identity.

For example:

```text
engineering.security#3
```

can identify clause 3 within the stable section.

Then a current citation such as:

```text
2.5.3
```

can resolve to that internal identity.

History can then be obtained through Git.

A useful future abstraction:

```text
LawHistory {
    current
    introduced
    modifications[]
    authors[]
}
```

The first implementation can simply map source locations into Git history without building a sophisticated history database.

---

# 38. Detecting Clause Changes

A clause's text should be tracked independently where possible.

A change from:

```text
3. Credentials discovered in source must be treated as compromised.
```

to:

```text
3. Credentials discovered in source must immediately be treated as compromised.
```

should be detectable as a law change.

The implementation should eventually support identifying:

```text
unchanged law
modified law
new law
removed law
```

rather than treating every compilation as an entirely new lawbook.

---

# 39. Git Integration

Git should remain the historical source of truth.

AgentLaws should use Git for:

* commit identity
* commit revision
* file history
* blame information
* change history
* optional signing

Avoid building a second history database unless there is a demonstrated need.

The core principle is:

> **AgentLaws adds structure to Git history; it does not replace Git history.**

---

# 40. Prompt Governor Workflow

The future Prompt Governor workflow should build on normal Git changes.

A conceptual workflow:

```text
Governor proposes amendment
        ↓
source files change
        ↓
alaws compile
        ↓
validation
        ↓
review
        ↓
signature
        ↓
git commit
```

The Governor role can apply equally to:

* human maintainers
* autonomous agents
* agent-assisted maintainers
* groups of agents

The core compiler should remain neutral about who is performing the governance.

---

# 41. Human and Agent Governors

The future system may support multiple Governors discussing changes.

For example:

```text
Governor A:
  Proposes a change to engineering.security.

Governor B:
  Questions the rationale.

Governor C:
  Suggests an alternative wording.

Governor A:
  Revises the law.

AgentLaws:
  Compiles and records the resulting state.
```

The important architectural decision is that these discussions eventually modify source files and produce compiled lawbooks.

The governance conversation should not become an opaque datastore disconnected from the source lawbook.

---

# 42. Precedence and Conflicts Are Deferred

AgentLaws does not currently need a formal precedence engine.

Do not implement:

```text
precedence weights
rule inheritance
automatic conflict resolution
exception semantics
scope semantics
```

for v1.

Prompt Governors can express whatever organizational or legal structure they currently need in the commentary and laws themselves.

The architecture should leave room for richer semantics later, but should not force those semantics into the initial IR.

---

# 43. Applicability Is Deferred

Do not add fields such as:

```text
environment
operation
actor
scope
resource
```

to the core metadata in v1.

The current requirement is simply:

```text
section has an ID
lawbook has ordered sections
laws have canonical numbers
```

Applications can select laws according to their own needs.

Later, AgentLaws may develop a richer applicability model based on real usage rather than speculation.

---

# 44. Validation Philosophy

The compiler should be conservative about structure and conservative about semantics.

It should be confident about:

```text
file exists
metadata exists
ID is present
ordering is valid
markers exist
laws are numbered
citations resolve
```

It should not pretend to know:

```text
whether two laws conflict
whether a law is reasonable
whether an exception is complete
whether a rule is legally superior
```

Those are governance judgments rather than compiler facts.

---

# 45. Testing Strategy

The compiler is a deterministic transformation and should be heavily tested.

Tests should cover:

## Parser

* valid frontmatter
* invalid frontmatter
* missing metadata
* duplicate markers
* missing markers
* Markdown parsing

## Ordering

* valid ordering
* missing file
* duplicate ordering entry
* unordered source file
* malformed path

## Laws

* valid numbered lists
* missing laws
* unnumbered content
* mixed content
* empty laws section

## Numbering

* section numbering
* clause numbering
* movement of sections
* numbering stability within a source revision

## Resolution

* valid citation
* invalid citation
* section lookup
* source lookup

## Compilation

* deterministic output
* stable hashes
* provenance generation

## Rendering

* HTML output
* PDF output
* commentary preservation
* law numbering preservation

## Signing

* canonical input
* signature verification
* tamper detection

---

# 46. Golden Tests

The Markdown source format is particularly well suited to golden tests.

For each fixture:

```text
fixtures/
    basic/
        alaws.toml
        principles.md
        security.md
        expected/
            lawbook.json
            output.html
```

The test compiles the fixture and compares it against expected output.

This will make format evolution much easier to manage.

---

# 47. Determinism Testing

The compiler should be tested for reproducibility.

Given identical source:

```text
compile
compile
compile
```

the canonical lawbook representation should be identical.

Renderer output may legitimately contain timestamps or other metadata, so the renderer should either:

* receive deterministic compilation metadata, or
* clearly separate nondeterministic presentation metadata from the canonical representation.

The signed object must remain deterministic.

---

# 48. Security Model

AgentLaws is a governance and provenance system, not a security boundary by itself.

A law saying:

```text
Agents must not reveal secrets.
```

does not technically prevent an agent from revealing secrets.

The value comes from:

```text
law → prompt context → agent decision → citation → audit trail
```

rather than from pretending that Markdown rules constitute runtime enforcement.

Future integrations can enforce particular laws mechanically where appropriate.

---

# 49. Tamper Detection

The provenance system should make unauthorized changes detectable.

The desired model is approximately:

```text
source state
    ↓
canonical lawbook
    ↓
hash
    ↓
signed manifest
```

A modified lawbook should therefore fail verification.

This does not prevent someone from modifying source files; it provides a mechanism for determining whether a compiled governed state matches the state that was signed.

---

# 50. Provenance in HTML and PDF

HTML and PDF should contain provenance metadata where practical.

Potential information:

```text
Lawbook
Revision
Compilation timestamp
AgentLaws version
Git revision
Compiler identity
Signature reference
```

The complete machine-readable provenance should remain available from the canonical manifest.

The visual document should not be overloaded with cryptographic implementation details.

---

# 51. API Surface Philosophy

The Go API should remain small.

The primary concepts should be things such as:

```text
Load
Compile
Lawbook
Section
Law
Resolve
History
Laws
Render
```

Avoid exposing internal parser or renderer implementation details as public API prematurely.

The goal is to let applications ask questions such as:

```go
book.Resolve("2.5.3")
book.Section("engineering.security")
book.Laws(...)
book.History("2.5.3")
```

without requiring them to understand the filesystem parser.

---

# 52. CLI vs Library Responsibilities

The CLI should be an interface to the core engine.

Do not implement business logic in shell commands that is absent from the library.

Preferred architecture:

```text
                 Go core
                /   |   \
               /    |    \
            CLI     UI   Application
```

rather than:

```text
CLI
 └── independent implementation

UI
 └── independent implementation

Library
 └── independent implementation
```

This is essential for maintaining consistent semantics across all interfaces.

---

# 53. Local Server Architecture

The local web UI can be served from the Go application.

Conceptually:

```text
alaws watch
    ↓
Go HTTP server
    ├── Lawbook API
    ├── diagnostics API
    ├── ordering update API
    └── static Preact assets
```

The UI can then operate entirely against the local process.

No hosted AgentLaws service is required for the core functionality.

The served UI follows the VS Code theming requirement in §28 — the Go server has no role in
styling beyond serving the static assets; all visual theming lives in the embedded `web/` app.

---

# 54. File Watching

The watcher should observe:

```text
alaws.toml
*.md
*.mdx
```

and any other supported source files.

On change:

```text
debounce
validate
compile
notify UI
```

A small debounce window is useful to avoid recompiling whiltor is saving a file in multiple operations.

---

# 55. Incremental Compilation

Initial implementation should favor simplicity.

Possible v1:

```text
any source change
    ↓
recompile entire cluster
```

Later:

```text
changed file
    ↓
reparse affected section
    ↓
rebuild affected lawbook structures
    ↓
rerender affected outputs
```

Do not implement incremental compilation until real lawbook sizes demonstrate a need.

---

# 56. Cluster Discovery

The compiler should be able to discover llusters in a repository.

A cluster is identified by:

```text
alaws.toml
```

The initial implementation can recursively search relevant paths while avoiding:

```text
.git
node_modules
vendor
build
dist
```

or other obviously irrelevant/generated directories.

An explicit cluster path should also be supported.

For example:

```bash
alaws compile ./payments
```

should compile the particular lawbook rather than searching the entire repository.

---

# 57. Multiple-Cluster Compilation

A repository-level command can eventually compile all discovered clusters:

```bash
alaws compile
```

while a targeted command can compile one:

```bash
alaws compile ./payments
```

The output should clearly identify which cluster failed if multiple clusters are being compiled.

---

# 58. Configuration Evolution

Keep `alaws.toml` intentionally small in v1.

Potential future fields may include:

```text
storage
output
rendering
signing
UI
```

but do not add fields before they are needed.

The ordering mechanism is the primary configuration feature.

---

# 59. File Naming

Both `.md` and `.mdx` should be considered supported source formats.

The implementation should not require MDX semantics simply because the project may have historically used the name.

The parser only needs:

```text
frontmatter
Markdown
AgentLaws structural comments
```

If arbitrary JSX/MDX execution becomes useful in the future, that can be introduced explicitly.

---

# 60. Why Not Execute MDX?

There is currently no requirement for arbitrary JSX execution.

Executing code while compiling governance material introduces:

* additional dependencies
* more complicated failure modes
* security implications
* harder reproducibility
* less predictable source semantics

The first implementation should treat Markdown as data and structure.

---

# 61. Future Richer Law Semantics

The internal IR should leave space for later concepts such as:

```text
Rule
Exception
Permission
Requirement
Prohibition
Condition
Applicability
Precedence
Reference
```

But v1 should represent these simply as law text.

The architecture should make it possible to add these later without forcing them into the first version of the file format.

---

# 62. Future Governance Operations

Potential future commands:

```bash
alaws propose
alaws amend
alaws review
alaws approve
alaws diff
alaws blame
alaws history
```

These would support Prompt Governors working with the lawbook as a living governance artifact.

Their implementation should build on:

```text
stable IDs
law citations
Git history
compiled revisions
signed provenance
```

rather than introducing an independent governance database.

---

# 63. Future Agent Governor Interaction

A later Agent Governor system could allow Governors to communicate in terms of precise law references:

```text
Governor A:
Amend engineering.security#3.

Governor B:
Why?

Governor A:
The current wording prevents incident remediation.

Governor B:
Proposed exception conflicts with another rule.

Governor A:
Revised amendment attached.

AgentLaws:
Compilation successful.
```

This is deliberately future-facing.

The current implementation needs only to provide the precise objects these conversations can reference.

---

# 64. Suggested Initial Milestones

A practical implementation sequence:

## Milestone 1 — Source parser

Implement:

```text
alaws.toml
frontmatter
commentary marker
laws marker
Markdown extraction
```

Deliver:

```text
Lawbook AST
```

---

## Milestone 2 — Compiler and validation

Implement:

```text
ordering
file discovery
diagnostiection IDs
title validation
law parsing
canonical numbering
```

Deliver:

```text
validated Lawbook IR
```

---

## Milestone 3 — Deterministic output

Implement:

```text
canonical JSON/IR representation
HTML renderer
```

Deliver:

```text
deterministic compiled lawbook
```

---

## Milestone 4 — CLI

Implement:

```bash
alaws compile
alaws list
alaws resolve
```

Deliver a useful command-line workflow.

---

## Milestone 5 — Agent extraction

Implement Go APIs for:

```text
section lookup
law look selection
law rendering
citation resolution
```

Deliver the first useful runtime integration.

---

## Milestone 6 — Git provenance

Implement:

```text
Git revision
Git identity
history
blame
manifest
hashing
```

Deliver traceability from citation to source history.

---

## Milestone 7 — Signing

Implement:

```text
canonical representation hashing
signature creation
signature verification
```

Deliver tamper-evident compiled lawbooks.

---

## Milestone 8 — Live compilation

Implement:

```bash
watch
```

with file watching and HTML refresh.

---

## Milestone 9 — Preact UI

Implement:

```text
navigation
diagnostics
section view
ordering editor
live refresh
```

The UI should initially be a thin frontend over the Go compiler.

---

## Milestone 10 — PDF and presentation polish

Add:

```text
PDF rendering
provenance presentation
print-friendly styling
```

Only after the semantic/compiler layer is stable.

---

# 65. Suggested Repository Structure

A starting repository could look like:

```tagentlaws/
├── cmd/
│   └── alaws/
│       └── main.go
│
├── internal/
│   ├── cli/
│   ├── compiler/
│   ├── discovery/
│   ├── lawedit/
│   ├── model/
│   ├── numbering/
│   ├── ordering/
│   ├── parser/
│   ├── provenance/
│   ├── renderer/
│   │   ├── html/
│   │   └── pdf/
│   ├── resolver/
│   ├── signing/
│   ├── template/
│   ├── validator/
│   ├── watcher/
│   └── server/
│
├── pkg/
│   └── alaws/
│       ├── model/
â   ├── compile/
│       ├── load/
│       ├── resolve/
│       └── render/
│
├── web/
│   └── ...
│
├── fixtures/
│   └── ...
│
├── tests/
│   └── ...
│
├── README.md
├── IMPLEMENTATION.md
├── go.mod
└── ...
```

This structure is illustrative rather than normative.

The main architectural requirement is separation between:

```text
parsing
validation
IR
compilation
rendering
provenance
signing
UI
```

-. First Version Definition of Done

A useful v1 should be able to take:

```text
alaws.toml
+
N Markdown files
```

and deterministically produce:

```text
validated Lawbook IR
+
canonical law numbers
+
human-readable HTML
+
optional PDF
+
machine-readable provenance
```

It should also be able to answer:

```text
What is law 2.5.3?

Which section contains it?

Which source file contains it?

What is the section's stable ID?

What revision produced it?

Who changed it?

```

And an application should be able to retrieve laws such that an agent can receive:

```text
2.5.1 ...
2.5.2 ...
2.5.3 ...
```

and return citations such as:

```text
Laws:
2.5.1
2.5.3
```

That is enough to establish the core AgentLaws model.

---

# 67. Architectural Summary

The core implementation can ultimately be summarized as:

```text
                      Markdown + TOML
                             │
                             ▼
                       Parser
                             │
                             ▼
              Lawbook IR
                             │
          ┌──────────────────┼──────────────────┐
          │                  │                  │
          ▼                  ▼                  ▼
       Validator          Numberer           Resolver
          │                  │                  │
          └──────────────────┼───────────â─────┘
                             ▼
                         Compiler
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                 │
           ▼                 ▼                 ▼
         HTML               PDF          Agent extraction
           │                 │                 │
           └─────────────┼─────────────────┘
                             ▼
                       Provenance
                             │
                       Hash / Sign
                             │
                             ▼
                            Git
```

The most important implementation boundary is the **Lawbook IR**.

Everything else should be a consumer or transformation of that representation.

---

# 68. Principles to Preserve During Implementation

Wheplementation decisions become ambiguous, prefer the option that preserves these properties:

1. **The source remains human-readable.**
2. **Ordering is explicit.**
3. **Directories remain purely organizational.**
4. **The compiler is deterministic.**
5. **Stable IDs are separate from presentation numbers.**
6. **Commentary and laws remain ordinary Markdown content.**
7. **The compiled lawbook is one coherent artifact.**
8. **Agent context is extracted from that artifact rather than becoming a second source of truth.**
9. **Every law citation is resolvable to source and history.**
10. **Git remains the historical system of record.**
11. **Compilation is attributable and eventually cryptographically verifiable.**
12. **Semantic complexity is added only when real governance use cases require it.**

The first release should therefore feel less like a policy programming language and more like a **compiler and version-control system for an organization's agent lawbooks**.

That foundation can later support much richer Prompt Governor workflows without forcing those assumptions into the initial format.

