# ✍️ Go Prompt Kit

[![CI](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/shouni/go-prompt-kit.svg)](https://pkg.go.dev/github.com/shouni/go-prompt-kit)
[![Status](https://img.shields.io/badge/Status-Completed-brightgreen)](#)

## 🚀 概要 (About) - AI 連携のインプットからアウトプットまでを一気通貫で

**Go Prompt Kit** は、AI（Gemini 等）へのプロンプト管理から、返ってきたレスポンス（Markdown / JSON）の美しいドキュメント化までをサポートする Go 言語向けツールキットです。

「プロンプト構築」と「洗練されたドキュメント配信」を組み合わせることで、AI 連携アプリケーションの開発効率と保守性を最大化します。

---

## ✨ 提供機能 (Features)

### 📂 [prompts] プロンプトエンジン

* **📦 Dynamic Resource Loader**: `embed.FS` からプロンプトを自動スキャンし、モード名へマッピング（再帰・拡張子フィルタ対応）。
* **🛠 Template-based Builder**: `text/template` にデータを注入して動的にプロンプトを生成。
* **🧱 Partial Templates**: 共通の指示を部品として切り出し、複数モードから共有。
* **🔧 Custom Functions**: 独自のテンプレート関数を登録し、本文と partial の双方から利用。
* **🎯 Default Mode**: モード未指定・未知のモードのフォールバック先を指定。
* **🔍 Expand**: データなしで partial 展開済みの本文を取得。カタログ表示や本文の検査に。
* **🛡 Collision Detection**: 名前の衝突・空ファイル・定義の重複を初期化時に検知。

### 📝 [frontmatter] プロンプト先頭のメタデータ

* **✂️ Split**: `---` で挟んだ front matter を本文から切り離し、メタデータが AI への指示に紛れ込むのを防止。
* **🔌 Pluggable Decode**: YAML ライブラリを固定せず、`yaml.Unmarshal` などを `UnmarshalFunc` として受け取る設計。**このパッケージの依存は標準ライブラリのみ**。
* **👀 Invisible Diff Normalization**: BOM と CRLF を判定前に揃えるため、「front matter を書いたのに認識されない」が起きません。

### 📡 [htmldoc] ドキュメント配信エンジン

* **📑 Markdown to HTML**: AI のレスポンスを、スタイル済みの完全な HTML ドキュメントへ変換。
* **🧾 JSON to HTML**: 構造化出力を、呼び出し側が指定した `html/template` で任意の形にHTMLフラグメント化。スキーマやテンプレートはライブラリ側では関知しない汎用設計。
* **🎨 Style-Injected Rendering**: 組み込みの CSS とテンプレートで即座に成果物を出力。差し替えも、既定を保った追記も可能。
* **🧩 Modular Architecture**: Converter と Renderer が `ports` のインターフェース越しに分離され、片方だけ差し替え可能。
* **🌲 AST-based Title Extraction**: 構文木からタイトルを抽出するため、コードブロック内の `#` や setext 形式も正しく判定。

---

## 🔰 使い方 (Usage)

### プロンプトを組み立てる

```go
//go:embed prompts/prompt_*.md
var promptFiles embed.FS

// embed.FS の読み込みと Builder の構築をまとめて行います
builder, err := prompts.LoadFS(promptFiles, "prompts", "prompt_")
if err != nil {
    return err
}

prompt, err := builder.Build("review", struct{ Diff string }{Diff: diff})
```

モード名はファイル名から接頭辞と拡張子を除いたものになります
（`prompts/prompt_review.md` → `review`）。

言語別ディレクトリのように階層を持たせる場合は再帰読み込みを使います。
モード名は `prompts/en/rock.md` → `en/rock` のように相対パスになります。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts", "", prompts.WithRecursive())
```

`_` で始まるファイルは partial として登録され、`Build` の対象にはなりません。
複数モードで共有する指示は partial に切り出せます。

```text
prompts/_output.md   → {{template "_output" .}} で参照する部品
prompts/review.md    → モード "review"
```

テンプレート関数は `WithFuncs` で登録します。partial からも呼び出せます。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts", "",
    prompts.WithRecursive(),
    prompts.WithFuncs(template.FuncMap{"join": strings.Join}),
)
```

すべてのテンプレートは1つの名前空間を共有します。
複数のファイルが `{{define "同じ名前"}}` を持つ場合は、静かに上書きされる前に
`ErrDuplicateDefinition` として構築時に検出されます。

モード未指定や未知のモードを既定へ寄せる場合は `WithDefaultMode` を使います。
呼び出し側でモードの有無を判定する必要がなくなります。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts/outline", "",
    prompts.WithExtensions(".md"),
    prompts.WithDefaultMode("default"),
)

prompt, err := builder.Build("", data) // 未指定なので "default" が使われる
```

partial の接頭辞は `WithPartialPrefix` で変更でき、空文字を指定すると
partial 判定自体を行わず、全エントリがモードとして公開されます。

### 送るプロンプトの中身を確認する

`Expand` は partial を展開した本文を、`{{.Field}}` を評価せずに返します。
データを用意せずに「実際に送られるプロンプトの構造」を確認できるため、
プロンプトのカタログ表示や、本文に書かれた制約の検査に使えます。

```go
text, err := builder.Expand("review")
// 対象: {{.Target}}
// 出力形式: JSON      ← partial は展開済み
```

`Build` と同じ構文木から組み立てるので、結果は実際に送られる本文と構造的に一致します。
入れ子の partial は再帰的に解決されるため定義順に依存せず、循環参照は
`ErrCyclicTemplate` として検出されます。データの起点が変わる
`{{template "x" .Foo}}` は展開できないため `ErrNotExpandable` を返します。

### プロンプト先頭のメタデータ（front matter）を切り離す

モードの説明をプロンプト自身に持たせると、モードを足す作業がファイルを1つ置くだけで
済みます。ただしメタデータをそのままテンプレートへ渡すと、AI への指示の先頭に
YAML が紛れ込みます。`frontmatter` は、その切り離しだけを担います。

```go
files, err := resource.Load(assets.Prompts, "prompts", "", resource.WithExtensions(".md"))
bodies, fronts := frontmatter.SplitMap(files)

// メタデータの構造はアプリが決めます。解析関数も呼び出し側が渡します。
infos, err := frontmatter.DecodeMap[ModeInfo](fronts, yaml.Unmarshal)

builder, err := prompts.NewBuilder(bodies)
```

**メタデータの書式はこのパッケージでは解釈しません。** `UnmarshalFunc` は
`func(data []byte, v any) error` なので、`yaml.Unmarshal` も `json.Unmarshal` も
そのまま渡せます。YAML ライブラリを固定しないのは、その選択と乗り換えを利用側の
ペースで行えるようにするためです（固定すると、乗り換えのたびにこのモジュールの
リリースと足並みを揃える必要が生じ、移行の途中では 1 つのバイナリに 2 つの実装が載ります）。

終了の区切りとみなすのは `---` だけからなる行です。`----` のように文字数が違う行は
区切りとみなしません。区切り行とその行末の改行だけを取り除くので、区切りの直後の
空行は本文に残ります。

### Markdown を HTML ドキュメントへ変換する

```go
doc, err := htmldoc.New(
    htmldoc.WithConverterOptions(markdown.WithHardWraps(true)),
    htmldoc.WithLang("ja-jp"),
)
if err != nil {
    return err
}

// タイトルに空文字を渡すと、最初の見出しから自動抽出されます
// 出力は io.Writer へ直接書き出されます（http.ResponseWriter やファイルにも渡せます）
err = doc.Run(w, "", src)
```

### JSON を任意のテンプレートで HTML 化する

`WithConverter` で任意の `ports.Converter` を注入できます。
CSS は `WithRendererOptions` から指定します。`renderer.WithCSS` は既定の
スタイルシートを置き換え、`renderer.WithExtraCSS` は既定の後ろへ連結します
（既定の体裁を保ったまま独自の部品スタイルだけを足したい場合はこちら）。

```go
tpl := template.Must(template.New("fragment").Parse(fragmentHTML))

doc, err := htmldoc.New(
    htmldoc.WithConverter(jsondoc.New(tpl)),
    htmldoc.WithRendererOptions(renderer.WithCSS(myCSS)),
)
if err != nil {
    return err
}

err = doc.Run(w, "", reviewJSON)
```

---

## 🏗 プロジェクトレイアウト (Project Layout)

機能ごとに独立したモジュール構成を採用しており、必要な機能だけを選択して利用可能です。

```text
go-prompt-kit/
├── prompts/           # 【INPUT】モード管理・partial・テンプレート実行
├── frontmatter/       # 【INPUT】プロンプト先頭のメタデータの切り離し（依存なし）
├── htmldoc/           # 【OUTPUT】ドキュメント配信（Document = 変換 + レンダリングの実行）
│   ├── ports/         #   - 抽象インターフェース定義 (Converter/Renderer/Runner)
│   ├── markdown/      #   - Markdown→HTMLフラグメント変換・タイトル抽出
│   ├── jsondoc/       #   - JSON→HTMLフラグメント変換（テンプレートは呼び出し側が注入）
│   └── renderer/      #   - HTML レンダリング (CSS/Template)
└── resource/          # 【BASE】fs.FS からのアセット自動スキャン
```

---

## ⚠️ 補足 (Notes)

* `frontmatter.Split` は改行を LF へ揃え、先頭の BOM を取り除いて返します（front matter の有無にかかわらず）。どちらもエディタ上で見えないまま判定を外すため、判定前に揃えます。
* `htmldoc.Document` は入力形式に関知しません。Markdown か JSON かは注入する `ports.Converter` が決めます。
* `ports.Converter.ExtractTitle` は形式非依存です。`jsondoc.Converter` はこれを「トップレベルの `title` キーの取得」として実装しています（キーは `jsondoc.WithTitleKey` で変更可能）。
* `markdown.WithUnsafeHTML(true)` は Markdown 中の生 HTML をそのまま出力します。信頼できない入力に対しては有効化しないでください。同様に `renderer.WithCSS` の内容もエスケープされずに `<style>` へ挿入されます。

---

## 🤝 主な依存関係 (Dependencies)

* [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark): Markdown 解析・HTML変換エンジン
* `text/template`: Go 標準のテンプレートエンジン
* `io/fs`: 抽象化されたファイルシステムインターフェース
* `embed`: 静的アセットのバイナリ埋め込み

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
