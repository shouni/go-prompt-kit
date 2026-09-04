# ✍️ Go Prompt Kit

[![CI](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)](#)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://go.dev/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://go.dev/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/shouni/go-prompt-kit.svg)](https://pkg.go.dev/github.com/shouni/go-prompt-kit)

## 🚀 概要 (About) - プロンプトの組み立てと、応答の HTML 化。AI そのものは知らない

**Go Prompt Kit** は、AI 連携アプリの入口と出口を引き受けるツールキットです。入口は
`embed.FS` に置いたプロンプトをモード別に読み込んで実行すること、出口は返ってきた
Markdown / JSON を完全な HTML ドキュメントへ組み立てることです。

**AI SDK には依存せず、送信も受信も行いません。** どのモデルへどう送るかはアプリの選択です。
出力側もスキーマやテンプレートの中身には関知せず、JSON の構造は呼び出し側の
`html/template` が決めます。

---

## ✨ 提供機能 (Features)

* **モード別プロンプト** — ファイル名がそのままモード名になり、モードを足す作業はファイルを
  1 つ置くだけで済みます。共通の指示は partial（`_` 始まり）に切り出して複数モードから共有できます。
* **送る前に中身を確かめられる** — `Expand` はデータ無しで partial 展開済みの本文を返し、
  `Fields` はそのモードが data に要求するフィールドを列挙します。カタログ表示や検査に使えます。
* **front matter** — モードの説明をプロンプト自身に持たせられます。切り離しから `Builder` への
  登録までが 1 回の読み込みで済みます。
* **HTML ドキュメント化** — Markdown か JSON かは注入する `Converter` が決め、CSS も外枠のテンプレートも
  差し替えられます。出力は `io.Writer` へ直接書き出すので、`http.ResponseWriter` にもファイルにも渡せます。
* **構築後は不変** — `Builder` も `Document` も、起動時に一度作って HTTP ハンドラーから並行に使えます。

---

## 📦 パッケージ構成 (Package Structure)

入口（INPUT）と出口（OUTPUT）は互いに依存せず、片方だけを使えます。

| パッケージ | 役割 |
| --- | --- |
| `prompts` | 【INPUT】モード管理・partial・テンプレート実行（`Builder` / `LoadFS`） |
| `resource` | 【INPUT】`fs.FS` からのファイル読み込みとモード名の導出（`Load`） |
| `frontmatter` | 【INPUT】プロンプト先頭のメタデータの切り離し（`Split` / `Decode`。標準ライブラリのみ） |
| `htmldoc` | 【OUTPUT】変換とレンダリングを束ねる入口（`Document.Run`） |
| `htmldoc/ports` | 【OUTPUT】`Converter` / `TitledConverter` / `Renderer` / `Runner` の定義 |
| `htmldoc/markdown` | 【OUTPUT】Markdown → HTML フラグメント変換とタイトル抽出（goldmark） |
| `htmldoc/jsondoc` | 【OUTPUT】JSON → HTML フラグメント変換（テンプレートは呼び出し側が注入） |
| `htmldoc/renderer` | 【OUTPUT】フラグメントを完全な HTML へ組み立てる（CSS / 外枠テンプレート） |

各シンボルの引数・戻り値・オプションは
[pkg.go.dev](https://pkg.go.dev/github.com/shouni/go-prompt-kit) にあります。

---

## 🚦 使い方 (Usage)

プロンプトを読み込んで 1 本組み立て、その応答を HTML にするまでです。

```go
//go:embed prompts/prompt_*.md
var promptFiles embed.FS

// モード名はファイル名から接頭辞と拡張子を除いたもの（prompt_review.md → "review"）。
builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithPrefix("prompt_"))
if err != nil {
    return err
}

prompt, err := builder.Build("review", struct{ Diff string }{Diff: diff})
// …ここで AI へ送るのは呼び出し側です…

doc, err := htmldoc.New(htmldoc.WithConverterOptions(markdown.WithCJK(true)))
if err != nil {
    return err
}
// タイトルに空文字を渡すと、最初の見出しから自動抽出されます。
err = doc.Run(w, "", response)
```

階層化（`prompts/en/rock.md` → `en/rock`）は `WithRecursive`、モード未指定のフォールバックは
`WithDefaultMode`、テンプレート関数は `WithFuncs` です。JSON の応答を任意のテンプレートで
組み立てる場合は `htmldoc.WithConverter(jsondoc.New(tpl))` を渡します。

**踏むと高くつく点も、それぞれの godoc に書いてあります** — テンプレートの名前空間が 1 つで
`{{define}}` が衝突すること、partial の末尾の改行が本文の途中で空行になること、読み込み専用の
オプションを `NewBuilder` に渡すとエラーになること、`Fields` が `range` / `with` の内側を列挙しない
こと、front matter の書式は `UnmarshalFunc` に委ねること、`jsondoc` が JSON の数値を字面のまま
扱うこと、`renderer` の CSS と本文はエスケープされないこと。

---

## 🤝 依存関係 (Dependencies)

本体のコードが参照する外部モジュールは **1 つだけ**です。

* [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark): Markdown 解析・HTML 変換（`htmldoc/markdown` のみが使用）
* [`stretchr/testify`](https://github.com/stretchr/testify): テストのみ

`prompts` / `resource` / `frontmatter` は標準ライブラリ（`text/template` / `io/fs` / `embed`）だけで、
`htmldoc/jsondoc` も `encoding/json` と `encoding/json/jsontext` だけで動きます。

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
