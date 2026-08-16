# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`github.com/shouni/go-prompt-kit` is a **library module** (no `main` package, no binary). It has two halves:

- **INPUT** (`prompts`, `resource`) — load prompt templates from an `embed.FS` and render them with injected data before sending to an AI.
- **OUTPUT** (`htmldoc/...`) — turn an AI response (Markdown or JSON) into a complete, styled HTML document.

The two halves share no code and are used independently.

## Commands

```bash
go build ./...
go test ./...
go test -race ./...                       # what CI runs
go test ./... -cover
go test ./prompts -run TestBuilder_Build  # single test
go test ./htmldoc/markdown -run TestConverter_ExtractTitle -v

go vet ./...
test -z "$(gofmt -l .)"                   # CI fails on any unformatted file
golangci-lint run                         # config in .golangci.yml; CI pins v2.12.2
```

CI (`.github/workflows/ci.yml`) runs build/vet/gofmt/race-test, golangci-lint, and govulncheck on pushes and PRs to `main` and `develop`. Current work happens on `develop`.

## Architecture

### The `htmldoc` pipeline

`htmldoc/ports/interfaces.go` defines the only three abstractions; everything else is an
implementation or the wiring that joins them.

```
input bytes ──> Converter.Convert() ──> HTML fragment ──> Renderer.Render(w, fragment, lang, title) ──> full HTML doc
                        └── htmldoc.Document orchestrates both and resolves the title ──┘
```

- **Converter** (`htmldoc/markdown` for Markdown via goldmark, `htmldoc/jsondoc` for JSON via a
  caller-supplied `html/template`) produces a *fragment*, never a whole document. Each package
  names its implementation `Converter` and constructs it with `New`.
- **Renderer** (`htmldoc/renderer`) wraps a fragment in `template.html` and inlines `default.css`,
  both `go:embed`-ed in `htmldoc/renderer/data.go`. `WithTemplate` / `WithTemplateText` / `WithCSS`
  override either one; `WithExtraCSS` appends to whichever stylesheet is in effect (later wins, so
  callers can add component styles without copying the 270-line default).
- **`htmldoc.Document`** (`htmldoc/document.go`) is the only `ports.Runner` implementation and the
  package's entry point. `htmldoc.New` builds the Markdown converter and the embedded-asset renderer
  *only when neither is injected*, so `WithConverter` / `WithRenderer` let a JSON pipeline go through
  the same constructor. `Document.Run(w, title, input)` writes straight to an `io.Writer` — it never
  buffers the whole document on the caller's behalf. The title resolves in this order: explicit
  `title` argument → `Converter.ExtractTitle(input)` → the default title. `lang` and the default
  title come from `WithLang` / `WithDefaultTitle` (defaults `"ja-jp"` and `"Document"`).

`ports` stays a separate package because both the implementations (`jsondoc` asserts
`var _ ports.Converter`) and the consumer (`htmldoc`) reference the interfaces; merging them into
`htmldoc` would make `jsondoc` → `htmldoc` → `jsondoc` cycles possible.

`ExtractTitle` is on the interface for *all* converters. `markdown.Converter` parses with goldmark
and walks the AST for the first `ast.Heading`, so fenced and indented code blocks, setext headings,
and closing `#` sequences are all handled by the parser; `inlineText` recursively collects text from
inline children, which strips emphasis, links, and code spans down to their text. `jsondoc.Converter`
implements the same method by reading a configurable top-level JSON key (default `"title"`).

### The `prompts` / `resource` flow

Dependent repos load an `embed.FS` into a mode→prompt map and execute one mode at a time. `prompts.LoadFS` collapses the whole flow; `resource.Load` + `prompts.NewBuilder` remain available separately.

```go
//go:embed prompts/prompt_*.md
var files embed.FS

builder, _ := prompts.LoadFS(files, "prompts", "prompt_")
out, _     := builder.Build("summarize", data)
```

- `resource.Load` derives the *mode name* from the filename by stripping the prefix and the extension, and returns an error on mode-name collision. An empty `prefix` loads every file in the directory. It is non-recursive by default; `WithRecursive` walks subdirectories and makes the mode name the `/`-joined path relative to `rootDir` (`prompts/en/rock.md` → `en/rock`). `WithExtensions` filters by extension.
- **All templates share one namespace.** `prompts.NewBuilder` registers every entry as an associated template on a single root, so a mode body can pull in another entry with `{{template "_name" .}}` — including a data argument, since expansion is native `text/template`, not string substitution.
- Because the namespace is shared, two entries that `{{define}}` the same name would silently overwrite each other (last writer wins). `recordDefinitions` re-parses each entry **in isolation** to learn exactly which names it defines, and rejects cross-entry duplicates with `ErrDuplicateDefinition`. Isolation matters: scanning the shared root cannot distinguish "this entry defined it" from "an earlier entry defined it". A `{{template "x"}}` *reference* creates no namespace entry, so references never trip the check.
- Entries whose name's last path element starts with `DefaultPartialPrefix` (`_`) are **partials**: registered for reference but excluded from `Modes()` and rejected by `Build`. A map containing only partials is an error. `WithPartialPrefix` changes the prefix; passing `""` disables partial detection entirely so every entry becomes a mode.
- `WithDefaultMode` makes `Build` fall back to a named mode for any unregistered mode, including `""`. The default mode must itself be a registered non-partial mode or `NewBuilder` fails — this catches typos at construction. `Has` deliberately ignores the fallback and reports actual registration.
- `Expand` (`prompts/expand.go`) is the counterpart to `Build`: it returns the mode's **source** with partials inlined but `{{.Field}}` actions left untouched, so callers can show or inspect a prompt without having data. It walks the `text/template/parse` tree and splices referenced templates into `TemplateNode` positions, descending into `if`/`range`/`with` bodies. It never mutates the stored trees — branch nodes are value-copied and a fresh `ListNode` is built — because `Build` keeps using them. A reference that changes the data context (`{{template "x" .Foo}}`) is rejected with `ErrNotExpandable` rather than silently rebinding `.`; cycles give `ErrCyclicTemplate`.
- `NewBuilder` sets `Option("missingkey=error")` on the root (the option lives in the shared `common` struct, so it applies to every associated template). A template referencing a field absent from the data fails at `Build` time rather than emitting `<no value>`. `WithFuncs` registers custom template functions before parsing; they are available to modes and partials alike.
- `prompts.Option` covers both loading and building. `WithRecursive` / `WithExtensions` only take effect in `LoadFS` (they are forwarded to `resource.Load`); `WithFuncs` applies in both.
- Sentinel errors: `prompts.ErrUnknownMode`, `prompts.ErrEmptyTemplates`, `prompts.ErrDuplicateDefinition`, `builder.ErrUnsupportedMode`. The first two keep their pre-sentinel message text because consumer tests match on it.

## Conventions

- All doc comments, error messages, and log output are in **Japanese**, in `です／ます` style.
- Options use the functional-option pattern: an `options.go` per package exporting `WithXxx()` returning `Option func(*config)`, applied into an unexported `config` struct that the constructor consumes. Constructors take `opts ...Option` so adding options stays backward compatible.
- Tests are table-driven with `github.com/stretchr/testify` (`assert` / `require`) and named `TestType_Method`.
- Errors are wrapped with `%w` at every layer boundary.
## API stability

The module is tagged `v1.x` and is consumed by five sibling repositories under `~/GolandProjects`
(`ap-comp`, `ap-mv`, `ap-story`, `ap-voice`, `git-gemini-web`), all pinned to a released tag with no
`replace` directives. **Every one of them imports `prompts` and/or `resource` only** — nothing
consumes `htmldoc/...`. Renaming or changing the signature of anything exported from `prompts` or
`resource` breaks all five; prefer additive changes there (variadic options on existing
constructors, type aliases when a name has to change).

`htmldoc/...` was reshaped in place precisely because it had no consumers (it was `md/...`, with
`md/converter`, `md/jsonconverter`, `md/runner` and `md/builder` as separate packages). Before
assuming that freedom still holds, re-check for importers.

To verify compatibility against the real consumers without touching their files, build them through
a temporary workspace:

```bash
cat > /tmp/gpk.work <<'EOF'
go 1.26
use (
	/Users/kensukeshouni/GolandProjects/go-prompt-kit
	/Users/kensukeshouni/GolandProjects/ap-comp
)
EOF
cd /Users/kensukeshouni/GolandProjects/ap-comp && GOWORK=/tmp/gpk.work go test ./...
```
