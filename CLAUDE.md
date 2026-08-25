# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`github.com/shouni/go-prompt-kit` is a **library module** (no `main` package, no binary). It has two halves:

- **INPUT** (`prompts`, `resource`, `frontmatter`) — load prompt templates from an `embed.FS` and render them with injected data before sending to an AI.
- **OUTPUT** (`htmldoc/...`) — turn an AI response (Markdown or JSON) into a complete, styled HTML document.

The two halves share no code and are used independently.

## Commands

```bash
go build ./...
go test ./...
go test -race ./...                       # what CI runs
go test ./... -cover
go test ./prompts -run TestBuilder_Build  # single test
go test ./frontmatter -run TestSplit -v
go test ./htmldoc/markdown -run TestConverter_ExtractTitle -v

go vet ./...
test -z "$(gofmt -l .)"                   # CI fails on any unformatted file
golangci-lint run                         # config in .golangci.yml; CI pins v2.12.2
```

CI (`.github/workflows/ci.yml`) runs build/vet/gofmt/race-test, golangci-lint, and govulncheck on pushes and PRs to `main` and `develop`. Current work happens on `develop`.

## Architecture

### The `htmldoc` pipeline

`htmldoc/ports/interfaces.go` defines the only abstractions; everything else is an
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
  `title` argument → the converter's extracted title → the default title. `lang` and the default
  title come from `WithLang` / `WithDefaultTitle` (defaults `"ja-jp"` and `"Document"`).

`ports` stays a separate package because both the implementations (`jsondoc` asserts
`var _ ports.Converter`) and the consumer (`htmldoc`) reference the interfaces; merging them into
`htmldoc` would make `jsondoc` → `htmldoc` → `jsondoc` cycles possible.

**`ports.TitledConverter` exists so auto-titling does not parse the input twice.** `Convert` and
`ExtractTitle` each parse the whole input, so `Run(w, "", input)` — the documented way to let the
title be extracted — used to do the work twice. The optional interface adds
`ConvertWithTitle(input) (fragment, title, error)`; `Document.convert` type-asserts for it and
falls back to `Convert` + `resolveTitle` when a converter only implements `Converter`. Both bundled
converters implement it, and both assert `var _ ports.TitledConverter` — without the assertion a
signature drift would silently drop `Document` back onto the two-parse path. When a title *is*
given, neither extraction path runs at all. Measured on a ~300-paragraph document (`Document.Run`
with an empty title, before → after): 1.05ms → 0.59ms, 13,689 → 8,360 allocs, 1.99MB → 1.11MB.
`markdown.Converter` can do this because `goldmark.Markdown.Convert` is just
`Parser().Parse()` + `Renderer().Render()`, so `convert` calls the two halves itself and keeps the
AST for `firstHeadingText`. `htmldoc/titled_test.go` pins that both paths render identical output.

`ExtractTitle` is on the interface for *all* converters. `markdown.Converter` parses with goldmark
and walks the AST for the first `ast.Heading`, so fenced and indented code blocks, setext headings,
and closing `#` sequences are all handled by the parser; `inlineText` recursively collects text from
inline children, which strips emphasis, links, and code spans down to their text. `jsondoc.Converter`
implements the same method by scanning with an `encoding/json/jsontext` decoder — it walks
top-level member names, `SkipValue()`s the rest, and stops at the configurable key (default
`"title"`), so it never materializes the document. A side effect: a title followed by malformed
JSON still comes back (`Convert` errors on that same input anyway).

`markdown.WithCJK` turns on goldmark's `extension.CJK`, which drops a soft line break when the
characters on both sides are East Asian wide. Without it a Japanese paragraph written across source
lines keeps the `\n` in the HTML, and a browser renders that as a space *inside* a sentence. It is
opt-in only to stay consistent with the other extension options (`WithTypographer`,
`WithFootnotes`); the default `lang` being `"ja-jp"` is an argument for flipping it if the option
shape ever changes. It affects the renderer only — `ExtractTitle` walks the AST, so a multi-line
setext heading still gets its space either way.

**`jsondoc` decodes numbers as `json.Number`, not `float64`.** Through `float64` a large integer
renders as `1.234567890123e+12` and `0.30` becomes `0.3` — the document then shows a different
number than the input carried, which defeats the point of a document generator. `decode` uses a
`json.Decoder` with `UseNumber()`; because a `Decoder` (unlike `json.Unmarshal`) silently ignores
whatever follows the first value, it then requires the next `Token()` to be `io.EOF`. Templates
that need arithmetic must convert the `json.Number` through a registered func.

### The `frontmatter` package

Prompt files carry their own metadata in a leading `---` block (mode description, category, which
input kind the mode expects). That block must come off before the body is registered as a template,
or the YAML lands at the top of the instruction sent to the model.

`Split` owns exactly that cut and nothing else:

- The closing delimiter is a line that is *only* `---`. `----`, `--- yaml`, and ` ---` are not
  delimiters. An earlier hand-rolled copy in `ap-story` matched on `"\n---"` and mis-split on `----`.
- Only the delimiter line and the newline ending it are removed, so a blank line right after the
  delimiter stays in the body. (One of the hand-rolled copies ate that blank line; the standardized
  behavior keeps it.)
- Both return values are normalized (BOM stripped, CRLF → LF) whether or not front matter was
  found. All three hand-rolled copies normalized only the front-matter-present path, so a CRLF file
  without front matter came back unnormalized. A BOM or CRLF is invisible in an editor and silently
  disables front matter detection — that is the failure this normalization exists to prevent.

**The package does not parse the metadata, and takes no dependency to do so.** `Decode` /
`DecodeMap` accept an `UnmarshalFunc func(data []byte, v any) error`, which `yaml.Unmarshal` and
`json.Unmarshal` both satisfy as-is. This is deliberate: consumers pick and migrate their own YAML
library (`gopkg.in/yaml.v3` has been frozen at v3.0.1 since 2022 and its upstream repo was archived
in April 2025; the maintained successor is `go.yaml.in/yaml/v3`). If this module hard-depended on
one, every swap would need a release here plus a bump in each consumer, and mid-migration a single
binary would link two YAML parsers deciding the same front matter. A `Parser` type holding the
function is not an option either — Go methods cannot have type parameters, so `DecodeMap[T]` has to
be a package-level function.

`DecodeMap` walks keys in sorted order so that, with several malformed entries, the reported key
does not change between runs. Entries with no front matter keep their key and get `T`'s zero value.

### The `prompts` / `resource` flow

Dependent repos load an `embed.FS` into a mode→prompt map and execute one mode at a time.

```go
//go:embed prompts/prompt_*.md
var files embed.FS

builder, _ := prompts.LoadFS(files, "prompts", prompts.WithPrefix("prompt_"))
out, _     := builder.Build("summarize", data)
```

- `resource.Load(fsys, rootDir, opts...)` derives the *mode name* from the filename by stripping the extension, and returns an error on mode-name collision. It is non-recursive by default; `WithRecursive` walks subdirectories and makes the mode name the `/`-joined path relative to `rootDir` (`prompts/en/rock.md` → `en/rock`). `WithExtensions` filters by extension. `WithPrefix` limits loading to files with that prefix and strips it from the mode name — it is an option rather than a positional parameter because every real call site passed `""`.
- **`LoadFS` is the entry point; prefer it over hand-wiring.** It used to be dead API — all five consumers called `resource.Load` + `NewBuilder` separately, because each needs a per-entry transform between the two (front matter, or a metadata comment). `WithFrontMatter` closes that gap: `LoadFS` splits each entry with `frontmatter.SplitMap`, registers only the body, and keeps the raw front matter behind `Builder.FrontMatter(name)` / `FrontMatters()`. The manual `resource.Load` + `frontmatter.SplitMap` + `NewBuilder` path stays available for callers that need the intermediate maps.
- **Load-only options are rejected by `NewBuilder`.** `WithPrefix` / `WithRecursive` / `WithExtensions` / `WithFrontMatter` only mean something while reading files, and used to be silently ignored here. They now record their name in `config.loadOnly`, and `NewBuilder` fails with `ErrLoadOnlyOption` listing them. `LoadFS` consumes them itself and calls the unexported `newBuilder`, bypassing the check.
- **A partial keeps its trailing newline unless `WithTrimPartials` is set.** Files end with a newline, so a partial referenced *mid-body* inserts a blank line at that point — which in Markdown is a paragraph break. Referenced at the end of a body it makes no visible difference, so the mismatch surfaces only wherever a consumer happens to place one mid-body — which is usually a small minority of the references, and easy to miss without a golden of the rendered output. Trimming is opt-in rather than the default because the default would change what every existing caller renders. The trim runs after the empty-content check, so a newline-only partial still registers as a partial that outputs nothing.
- **All templates share one namespace.** Every entry is registered as an associated template on a single root, so a mode body can pull in another entry with `{{template "_name" .}}` — including a data argument, since expansion is native `text/template`, not string substitution.
- Because the namespace is shared, two entries that `{{define}}` the same name would silently overwrite each other (last writer wins). `parseEntry` parses each entry **in isolation** to learn exactly which names it defines and rejects cross-entry duplicates with `ErrDuplicateDefinition`; isolation matters because scanning the shared root cannot distinguish "this entry defined it" from "an earlier entry defined it". `attach` then moves those parse trees into the root with `AddParseTree`, so **each entry is parsed once**. Re-parsing into the root was not just wasted work — it made the error branch there unreachable, since the isolated parse hits the same syntax error first. A `{{template "x"}}` *reference* produces a tree-less entry, which both functions skip: it never trips the duplicate check and never clobbers a real definition in the root.
- Entries whose name's last path element starts with `DefaultPartialPrefix` (`_`) are **partials**: registered for reference but excluded from `Modes()` and rejected by `Build`. A map containing only partials is an error. `WithPartialPrefix` changes the prefix; passing `""` disables partial detection entirely so every entry becomes a mode. The predicate is exported as `IsPartial(name, prefix)` so callers filtering a raw map do not re-implement it — a consumer's own copy tested the whole key instead of `path.Base`, which would have diverged the moment it used `WithRecursive`.
- `WithDefaultMode` makes `Build` fall back to a named mode for any unregistered mode, including `""`. The default mode must itself be a registered non-partial mode or `NewBuilder` fails — this catches typos at construction. `Has` deliberately ignores the fallback and reports actual registration.
- `Expand` (`prompts/expand.go`) is the counterpart to `Build`: it returns the mode's **source** with partials inlined but `{{.Field}}` actions left untouched, so callers can show or inspect a prompt without having data. It walks the `text/template/parse` tree and splices referenced templates into `TemplateNode` positions, descending into `if`/`range`/`with` bodies. It never mutates the stored trees — branch nodes are value-copied and a fresh `ListNode` is built — because `Build` keeps using them. A reference that changes the data context (`{{template "x" .Foo}}`) is rejected with `ErrNotExpandable` rather than silently rebinding `.`; cycles give `ErrCyclicTemplate`.
- `newBuilder` sets `Option("missingkey=error")` on the root (the option lives in the shared `common` struct, so it applies to every associated template). A template referencing a field absent from the data fails at `Build` time rather than emitting `<no value>`. `WithFuncs` registers custom template functions before parsing; they are available to modes and partials alike.
- **`Builder` is immutable after construction and safe for concurrent use.** Four consumers call `Build` from HTTP handlers. `prompts/concurrent_test.go` pins that guarantee — it is why `Modes` clones and `FrontMatters` returns a copy — and only means anything under `-race`.
- Sentinel errors: `prompts.ErrUnknownMode`, `prompts.ErrEmptyTemplates`, `prompts.ErrDuplicateDefinition`, `prompts.ErrLoadOnlyOption`, `prompts.ErrCyclicTemplate`, `prompts.ErrNotExpandable`, `frontmatter.ErrNoUnmarshalFunc`, `resource.ErrNotDirectory`. The first two keep their pre-sentinel message text because consumer tests match on it.

## Conventions

- **Error text**: sentinel errors are English with a package prefix (`review: diff is empty`) so a deeply wrapped error still names its origin; the context added by `fmt.Errorf` wrapping is Japanese. Existing English wrap text is not being retrofitted — apply the rule to code you touch.
- All doc comments, error messages, and log output are in **Japanese**, in `です／ます` style.
- Options use the functional-option pattern: an `options.go` per package exporting `WithXxx()` returning `Option func(*config)`, applied into an unexported `config` struct that the constructor consumes. Constructors take `opts ...Option` so adding options stays backward compatible.
- Tests are table-driven with `github.com/stretchr/testify` (`assert` / `require`) and named `TestType_Method`.
- Errors are wrapped with `%w` at every layer boundary.
- Direct dependencies are deliberately minimal: `goldmark` (for `htmldoc/markdown`) and `testify`
  (tests only). `prompts`, `resource`, and `frontmatter` use the standard library alone. Before
  adding a `require`, check whether the consumer can inject the thing instead — `frontmatter`'s
  `UnmarshalFunc` is the pattern to follow.

## API stability

The module is tagged `v1.x` and is consumed by the sibling repositories checked out under
`~/GolandProjects`, all pinned to a released tag with no `replace` directives. **Every one of them
imports `prompts` and/or `resource` only** — nothing consumes `htmldoc/...`. Renaming or changing the signature of anything
exported from `prompts` or `resource` breaks all of them at once; prefer additive changes there
(variadic options on existing constructors, type aliases when a name has to change).

`htmldoc/...` was reshaped in place precisely because it had no consumers (it was `md/...`, with
`md/converter`, `md/jsonconverter`, `md/runner` and `md/builder` as separate packages). Before
assuming that freedom still holds, re-check for importers.

To verify compatibility against the real consumers without touching their files, build them through
a temporary workspace:

```bash
cat > /tmp/gpk.work <<'EOF'
go 1.26.6
use (
	/Users/kensukeshouni/GolandProjects/go-prompt-kit
	/Users/kensukeshouni/GolandProjects/ap-story
)
EOF
cd /Users/kensukeshouni/GolandProjects/ap-story && GOWORK=/tmp/gpk.work go test ./...
```

Add the other consumers to the `use` block to cover them in one run. The `go` directive has to be
at least as new as every module listed, or the workspace is rejected before anything builds.
