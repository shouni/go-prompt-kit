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

---

## 📂 プロンプトを書くときの罠 (Authoring)

**テンプレートは 1 つの名前空間を共有します。** 複数のファイルが `{{define "同じ名前"}}` を持つと
後勝ちで静かに上書きされる——ところを、構築時に `ErrDuplicateDefinition` として止めます。

**partial を本文の途中へ差し込むなら `WithTrimPartials` を検討してください。** ファイルは改行で
終わるため、そのままだと差し込んだ位置にだけ空行が入ります。空行が段落の区切りになる形式
（Markdown など）ではそこだけ段落が分かれます。**末尾で参照している限り出力は変わらないので、
途中で使った箇所だけで表面化します。** 既定で取り除かないのは、既存の呼び出しの出力を
変えないためです。

**読み込み専用のオプションを `NewBuilder` に渡すとエラーになります。** `WithPrefix` /
`WithRecursive` / `WithExtensions` / `WithFrontMatter` は `LoadFS` にしか意味がありません。
黙って無視すると、絞り込んだつもりのまま全ファイルが登録されます（`ErrLoadOnlyOption`）。

**`Fields` の結果は「確かに要求されている」だけで、「これで全部」ではありません。**
`range` と `with` の本体は列挙しません。その内側では `.` が別の値を指すため、フィールド名を
data からの位置として報告できないからです。

**front matter の書式はこのライブラリでは解釈しません。** `yaml.Unmarshal` などを
`UnmarshalFunc` として渡してください。判定の前に BOM と CRLF を揃えるので、「front matter を
書いたのに認識されない」はここでは起きません。

---

## 📄 HTML にするときの罠 (Rendering)

**日本語の文書では `markdown.WithCJK(true)` を付けてください。** Markdown ではソフト改行が出力
HTML にもそのまま残り、ブラウザ上では空白として描画されます。単語を空白で区切る言語では正しい
挙動ですが、日本語では文の途中に空白が入ります。

```text
入力:   これは日本語の文章で、
        途中で改行しています。

無効:   <p>これは日本語の文章で、\n途中で改行しています。</p>  → 「、 途中で」と表示される
有効:   <p>これは日本語の文章で、途中で改行しています。</p>
```

取り除かれるのは改行の前後がともに全角文字である場合だけなので、英文が混ざっていてもそちらの
区切りは保たれます。

**`jsondoc` は JSON の数値を字面のまま扱います。** `float64` を経由すると桁の大きい整数が
`1.234567890123e+12` になり、`0.30` も `0.3` に丸められて、文書に載る数が入力と食い違います。
テンプレート側で数値として比較・計算する場合は、変換用の関数を `Funcs` で登録してください。

**自作の `Converter` では `ports.TitledConverter` の実装を検討してください。** タイトルを自動抽出
する場合、`Convert` と `ExtractTitle` をそれぞれ呼ぶと同じ入力を 2 回解析します。同梱の 2 つは
どちらも実装済みで、`Document` は実装があればそちらを優先します。

**エスケープされない場所が 3 つあります。** `renderer.WithCSS` で渡した CSS、外枠テンプレートの
`.Style` と `.Content`、そして `markdown.WithUnsafeHTML(true)` を有効にしたときの Markdown 中の
生 HTML です。いずれも信頼できる入力である前提で、利用者の入力を直接流し込む場所ではありません。

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
