# AGENTS.md — Guidelines for AI Agents

## Build & Test

```bash
make all             # build web UI + Go binary + run all tests
make build           # full build (web + Go)
make build-go        # Go-only build (skip web rebuild)
make test            # run all tests
make test-cover      # run with coverage report
make install         # install to ~/go/bin
make serve           # build and serve the web UI on localhost:8420
make watch           # build and watch fixtures/basic for changes
```

## Writing tests

- Use table-driven tests (`[]struct{name string; ...}` with `t.Run`)
- Use `t.TempDir()` for tests that need a file path
- No external test libraries — use stdlib `testing` only
- Each test should be independent and not rely on test execution order
- Name tests: `TestFunctionName_Scenario` (e.g., `TestRender_MissingVar`)

### What to test

- Template substitution (`internal/template`) — substitution, escaping, missing-var policies, malformed placeholders
- Parser (`internal/parser`) — frontmatter, commentary/laws markers, edge cases
- Ordering (`internal/ordering`) — insert, move, remove, parent-child derivation
- Law editing (`internal/lawedit`) — append, insert-after, remove, renumbering
- Validator (`internal/validator`) — every diagnostic code
- Resolver (`internal/resolver`) — valid/invalid citations, section lookup
- Compiler (`internal/compiler`) — deterministic output, stable hashes

### What NOT to test

- Don't test CLI cobra wiring (test the underlying `pkg/alaws` functions instead)
- Don't test third-party library behavior (goldmark, chroma, etc.)
- Don't test the web UI with Go tests (use the browser for that)

## Publishing to pkg.go.dev

### Best practices checklist

Before tagging a release, verify:

1. **`go.mod` exists** — module path is `github.com/shrsv/AgentLaws`
2. **LICENSE file exists** — at repo root (pkg.go.dev checks redistributability)
3. **Package doc comment** — first sentence is indexed by pkg.go.dev search; must be concise and keyword-rich
4. **Example test functions** — render as "Examples" section on pkg.go.dev (`Example()`, `ExampleBook_Resolve()`, etc. in `pkg/alaws/example_test.go`)
5. **All tests pass** — `make test`
6. **`go vet` clean** — `go vet ./...`
7. **Tagged version** — use semver (`v0.1.0`, `v0.2.0`, etc.)

### Release procedure

```bash
# 1. Run all checks
make release-check

# 2. Tag and push
make tag V=v0.1.0

# 3. Verify on pkg.go.dev (takes a few minutes to index)
# Visit: https://pkg.go.dev/github.com/shrsv/AgentLaws@v0.1.0
# Click "Request" if not yet indexed
```

### Versioning rules

- Follow semver: `vMAJOR.MINOR.PATCH`
- **v0.x.x** = experimental (current state); breaking changes allowed in minor bumps
- **v1.0.0+** = stable; breaking changes require major version bump
- Always tag from a clean working tree
- Write a brief release note for each tag (what changed, what's new)

### Godoc conventions

- Package doc comment: first sentence = one-line summary (indexed for search)
- All exported types and functions must have doc comments
- Example functions in `example_test.go` render on pkg.go.dev — add them for all public APIs
- Use `[TypeName]` bracket syntax for cross-references in doc comments (renders as links)
- Use `# Heading` in doc comments for godoc section headers
