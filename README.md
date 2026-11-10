# ✍️ Go Prompt Kit

[![CI](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)](#)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://go.dev/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://go.dev/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/shouni/go-prompt-kit.svg)](https://pkg.go.dev/github.com/shouni/go-prompt-kit)

English | [日本語](README.ja.md)

## 🚀 About - Assembles prompts and turns responses into HTML. Knows nothing about the AI itself

**Go Prompt Kit** is a toolkit that takes care of the entrance and the exit of an AI-backed app. The
entrance is loading prompts placed in an `embed.FS` per mode and executing them; the exit is
assembling the returned Markdown / JSON into a complete HTML document.

**It does not depend on any AI SDK, and it neither sends nor receives.** Which model gets what, and
how, is the app's choice. The output side does not care about schemas or template contents either:
the structure of the JSON is decided by the caller's `html/template`.

---

## ✨ Features

* **Prompts per mode** — the file name is the mode name, so adding a mode is just placing one
  file. Shared instructions can be cut out into partials (starting with `_`) and shared by several modes.
* **Inspect before sending** — `Expand` returns the body with partials expanded, without any data,
  and `Fields` lists the fields the mode requires from its data. Useful for catalog views and checks.
* **Front matter** — a prompt can carry its own mode description. Splitting it off and registering
  the body with the `Builder` take a single load.
* **HTML documents** — the injected `Converter` decides whether the input is Markdown or JSON, and
  both the CSS and the outer template can be replaced. Output is written straight to an `io.Writer`,
  so it can go to an `http.ResponseWriter` or a file.
* **Immutable after construction** — both `Builder` and `Document` can be built once at startup and
  used concurrently from HTTP handlers.

---

## 📦 Package Structure

The entrance (INPUT) and the exit (OUTPUT) do not depend on each other; either can be used alone.

| Package | Role |
| --- | --- |
| `prompts` | [INPUT] Mode management, partials and template execution (`Builder` / `LoadFS`) |
| `resource` | [INPUT] Reading files from an `fs.FS` and deriving mode names (`Load`) |
| `frontmatter` | [INPUT] Splitting the metadata off the top of a prompt (`Split` / `Decode`; standard library only) |
| `htmldoc` | [OUTPUT] Entry point tying conversion and rendering together (`Document.Run`) |
| `htmldoc/ports` | [OUTPUT] Definitions of `Converter` / `TitledConverter` / `Renderer` / `Runner` |
| `htmldoc/markdown` | [OUTPUT] Markdown → HTML fragment conversion and title extraction (goldmark) |
| `htmldoc/jsondoc` | [OUTPUT] JSON → HTML fragment conversion (the caller injects the template) |
| `htmldoc/renderer` | [OUTPUT] Assembles a fragment into a complete HTML document (CSS / outer template) |

The arguments, return values and options of every symbol are on
[pkg.go.dev](https://pkg.go.dev/github.com/shouni/go-prompt-kit).

---

## 🚦 Usage

From loading prompts and building one, to turning its response into HTML:

```go
//go:embed prompts/prompt_*.md
var promptFiles embed.FS

// The mode name is the file name without the prefix and extension (prompt_review.md → "review").
builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithPrefix("prompt_"))
if err != nil {
    return err
}

prompt, err := builder.Build("review", struct{ Diff string }{Diff: diff})
// …sending it to the AI is the caller's job…

doc, err := htmldoc.New(htmldoc.WithConverterOptions(markdown.WithCJK(true)))
if err != nil {
    return err
}
// An empty title makes it extracted from the first heading.
err = doc.Run(w, "", response)
```

Hierarchies (`prompts/en/rock.md` → `en/rock`) come from `WithRecursive`, a fallback for an
unspecified mode from `WithDefaultMode`, and template functions from `WithFuncs`. To assemble a JSON
response with your own template, pass `htmldoc.WithConverter(jsondoc.New(tpl))`.

**The costly pitfalls are documented in their godoc too** — there is a single template namespace, so
`{{define}}` names can collide; a partial's trailing newline becomes a blank line mid-body; passing a
load-only option to `NewBuilder` is an error; `Fields` does not list what is inside `range` / `with`;
the front matter format is left to an `UnmarshalFunc`; `jsondoc` keeps JSON numbers as written; and
the `renderer` does not escape the CSS or the body.

---

## 🤝 Dependencies

The library code references **only one** external module.

* [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark): Markdown parsing and HTML conversion (used only by `htmldoc/markdown`)
* [`stretchr/testify`](https://github.com/stretchr/testify): tests only

`prompts` / `resource` / `frontmatter` use only the standard library (`text/template` / `io/fs` / `embed`),
and `htmldoc/jsondoc` runs on `encoding/json` and `encoding/json/jsontext` alone.

---

## 📜 License

This project is released under the [MIT License](https://opensource.org/licenses/MIT).
