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

### 📡 [md] ドキュメント配信エンジン

* **📑 Markdown to HTML**: AI のレスポンスを、スタイル済みの完全な HTML ドキュメントへ変換。
* **🧾 JSON to HTML**: 構造化出力を、呼び出し側が指定した `html/template` で任意の形にHTMLフラグメント化。スキーマやテンプレートはライブラリ側では関知しない汎用設計。
* **🎨 Style-Injected Rendering**: 組み込みの CSS とテンプレートで即座に成果物を出力。差し替えも、既定を保った追記も可能。
* **🧩 Modular Architecture**: Converter, Renderer, Runner が分離され、特定のロジックのみを差し替え可能。
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

### Markdown を HTML ドキュメントへ変換する

```go
b, err := builder.New(
    builder.WithEnableHardWraps(true),
    builder.WithLang("ja-jp"),
)
if err != nil {
    return err
}

runner, err := b.BuildRunner()
if err != nil {
    return err
}

// タイトルに空文字を渡すと、最初の見出しから自動抽出されます
buf, err := runner.Run("", markdown)
```

### JSON を任意のテンプレートで HTML 化する

`WithConverter` で任意の `ports.Converter` を注入できます。
CSS は `WithRendererOptions` から指定します。`renderer.WithCSS` は既定の
スタイルシートを置き換え、`renderer.WithExtraCSS` は既定の後ろへ連結します
（既定の体裁を保ったまま独自の部品スタイルだけを足したい場合はこちら）。

```go
tpl := template.Must(template.New("fragment").Parse(fragmentHTML))

b, err := builder.New(
    builder.WithConverter(jsonconverter.New(tpl)),
    builder.WithRendererOptions(renderer.WithCSS(myCSS)),
)
if err != nil {
    return err
}

runner, err := b.BuildRunner()
if err != nil {
    return err
}

buf, err := runner.Run("", reviewJSON)
```

---

## 🏗 プロジェクトレイアウト (Project Layout)

機能ごとに独立したモジュール構成を採用しており、必要な機能だけを選択して利用可能です。

```text
go-prompt-kit/
├── prompts/           # 【INPUT】モード管理・partial・テンプレート実行
├── md/                # 【OUTPUT】ドキュメント配信
│   ├── ports/         #   - 抽象インターフェース定義
│   ├── converter/     #   - Markdown 解析・タイトル抽出
│   ├── jsonconverter/ #   - JSON→HTMLフラグメント変換（テンプレートは呼び出し側が注入）
│   ├── renderer/      #   - HTML レンダリング (CSS/Template)
│   ├── runner/        #   - 変換ワークフローの実行
│   └── builder/       #   - 具象インスタンスの構築・依存の注入
└── resource/          # 【BASE】fs.FS からのアセット自動スキャン
```

---

## ⚠️ 補足 (Notes)

* `runner.MarkdownToHTMLRunner` は `runner.DocumentRunner` の別名です。入力形式は注入する Converter が決めるため、新しいコードでは `DocumentRunner` / `NewDocumentRunner` を使用してください。
* `ports.Converter.ExtractTitleFromMarkdown` は名前に反して形式非依存です。`JSONConverter` はこれを「トップレベルの `title` キーの取得」として実装しています。互換性のため v1 では改名していません。
* `converter.WithUnsafeHTML(true)` は Markdown 中の生 HTML をそのまま出力します。信頼できない入力に対しては有効化しないでください。同様に `renderer.WithCSS` の内容もエスケープされずに `<style>` へ挿入されます。

---

## 🤝 主な依存関係 (Dependencies)

* [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark): Markdown 解析・HTML変換エンジン
* `text/template`: Go 標準のテンプレートエンジン
* `io/fs`: 抽象化されたファイルシステムインターフェース
* `embed`: 静的アセットのバイナリ埋め込み

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
