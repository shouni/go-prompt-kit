# ✍️ Go Prompt Kit

[![CI](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/go-prompt-kit/actions/workflows/ci.yml)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/shouni/go-prompt-kit.svg)](https://pkg.go.dev/github.com/shouni/go-prompt-kit)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)](#)

## 🚀 概要 (About) - AI 連携のインプットからアウトプットまでを一気通貫で

**Go Prompt Kit** は、AI（Gemini 等）へのプロンプト管理から、返ってきたレスポンス（Markdown / JSON）の美しいドキュメント化までをサポートする Go 言語向けツールキットです。

「プロンプト構築」と「洗練されたドキュメント配信」を組み合わせることで、AI 連携アプリケーションの開発効率と保守性を最大化します。

---

## ✨ 提供機能 (Features)

### 📂 [prompts] プロンプトエンジン

* **📦 Dynamic Resource Loader**: `embed.FS` からプロンプトを自動スキャンし、モード名へマッピング（再帰・拡張子フィルタ対応）。
* **🛠 Template-based Builder**: `text/template` にデータを注入して動的にプロンプトを生成。
* **🧱 Partial Templates**: 共通の指示を部品として切り出し、複数モードから共有。本文の途中へ差し込む場合は `WithTrimPartials` で末尾の改行を落とせます。
* **🔧 Custom Functions**: 独自のテンプレート関数を登録し、本文と partial の双方から利用。
* **🎯 Default Mode**: モード未指定・未知のモードのフォールバック先を指定。
* **🔍 Expand**: データなしで partial 展開済みの本文を取得。カタログ表示や本文の検査に。
* **📝 Front Matter**: `WithFrontMatter` を付けるだけで、切り離しから `Builder` への登録までを一度の読み込みで完了。
* **🔒 Concurrency-Safe**: 構築後の `Builder` は不変。起動時に一度作って HTTP ハンドラから並行に使えます。
* **🛡 Collision Detection**: 名前の衝突・空ファイル・定義の重複を初期化時に検知。

### 📝 [frontmatter] プロンプト先頭のメタデータ

* **✂️ Split**: `---` で挟んだメタデータを本文から切り離す処理そのもの。`prompts` を介さず単体でも使えます。
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
builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithPrefix("prompt_"))
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
builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithRecursive())
```

`_` で始まるファイルは partial として登録され、`Build` の対象にはなりません。
複数モードで共有する指示は partial に切り出せます。

```text
prompts/_output.md   → {{template "_output" .}} で参照する部品
prompts/review.md    → モード "review"
```

partial を**本文の途中**へ差し込む場合は `WithTrimPartials` を検討してください。
ファイルは改行で終わるのが普通なので、そのままだと差し込んだ位置にだけ空行が入り、
空行が段落の区切りを意味する形式（Markdown など）ではそこだけ段落が分かれます。
末尾で参照している限り出力に差は出ないため、本文の途中で使った箇所だけで表面化します。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithTrimPartials())
```

取り除くのは partial 末尾の改行だけです。モード本文の末尾や行末の空白は変わりません。
既定で取り除かないのは、末尾で参照している既存の呼び出しの出力を変えないためです。

テンプレート関数は `WithFuncs` で登録します。partial からも呼び出せます。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts",
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
builder, err := prompts.LoadFS(promptFiles, "prompts/outline",
    prompts.WithExtensions(".md"),
    prompts.WithDefaultMode("default"),
)

prompt, err := builder.Build("", data) // 未指定なので "default" が使われる
```

partial の接頭辞は `WithPartialPrefix` で変更でき、空文字を指定すると
partial 判定自体を行わず、全エントリがモードとして公開されます。
読み込んだマップを `Builder` に渡す前に自前で選り分ける場合は、判定を二重に書かず
`prompts.IsPartial(name, prompts.DefaultPartialPrefix)` を使ってください。

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
YAML が紛れ込みます。`LoadFS` に `WithFrontMatter` を指定すると、本文だけが
テンプレートとして登録され、切り離した front matter は `Builder` から取り出せます。

```go
builder, err := prompts.LoadFS(promptFiles, "prompts",
    prompts.WithExtensions(".md"),
    prompts.WithFrontMatter(),
)

// メタデータの構造はアプリが決めます。解析関数も呼び出し側が渡します。
infos, err := frontmatter.DecodeMap[ModeInfo](builder.FrontMatters(), yaml.Unmarshal)

prompt, err := builder.Build("review", data) // 本文だけが実行されます
```

読み込みと解析を自分で組み立てる場合は `frontmatter` を直接使えます。

```go
files, err := resource.Load(assets.Prompts, "prompts", resource.WithExtensions(".md"))
bodies, fronts := frontmatter.SplitMap(files)
builder, err := prompts.NewBuilder(bodies)
```

**メタデータの書式はこのパッケージでは解釈しません。** `UnmarshalFunc` は
`func(data []byte, v any) error` なので、`yaml.Unmarshal` も `json.Unmarshal` も
そのまま渡せます。YAML ライブラリを固定しないのは、その選択と乗り換えを利用側の
ペースで行えるようにするためです（固定すると、乗り換えのたびにこのモジュールの
リリースと足並みを揃える必要が生じ、移行の途中では 1 つのバイナリに 2 つの実装が載ります）。

切り離しの規則は次の3点です。

* 区切りとみなすのは `---` **だけ**からなる行です。`----` のように文字数が違う行は区切りではありません。
* 取り除くのは区切り行とその行末の改行だけなので、区切りの直後の空行は本文に残ります。
* 戻り値は front matter の有無にかかわらず、改行を LF へ揃え先頭の BOM を取り除いたものです。どちらもエディタ上で見えないまま判定を外すため、判定の前に揃えます。

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

* 読み込み専用のオプション（`WithPrefix` / `WithRecursive` / `WithExtensions` / `WithFrontMatter`）を `NewBuilder` に渡すと `ErrLoadOnlyOption` になります。黙って無視すると、絞り込んだつもりのまま全ファイルが登録されるためです。
* `htmldoc.Document` は入力形式に関知しません。Markdown か JSON かは注入する `ports.Converter` が決めます。
* `ports.Converter.ExtractTitle` は形式非依存です。`jsondoc.Converter` はこれを「トップレベルの `title` キーの取得」として実装しています（キーは `jsondoc.WithTitleKey` で変更可能）。
* `markdown.WithUnsafeHTML(true)` は Markdown 中の生 HTML をそのまま出力します。信頼できない入力に対しては有効化しないでください。同様に `renderer.WithCSS` の内容もエスケープされずに `<style>` へ挿入されます。

---

## 🤝 依存関係 (Dependencies)

外部モジュールへの直接依存は **1 つだけ**です。

* [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark): Markdown 解析・HTML 変換（`htmldoc/markdown` のみが使用）

`prompts` / `resource` / `frontmatter` は標準ライブラリ（`text/template` / `io/fs` / `embed`）だけで動きます。
とくに `frontmatter` が YAML ライブラリを持たないのは意図的な設計です（上記参照）。
テストのみ [`stretchr/testify`](https://github.com/stretchr/testify) を使用します。

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
