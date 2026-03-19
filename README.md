# ✍️ Go Prompt Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - AI 連携のインプットからアウトプットまでを一気通貫で。

**Go Prompt Kit** は、AI（Gemini 等）へのプロンプト管理から、AI から返ってきた Markdown レスポンスの美しいドキュメント化までをサポートする Go 言語向けツールキットです。

「プロンプト構築」と「洗練されたドキュメント配信（Cast）」を組み合わせることで、AI 連携アプリケーションの開発効率と保守性を最大化します。

---

## ✨ 提供機能 (Features)

### 📂 [prompts] プロンプトエンジン

* **📦 Dynamic Resource Loader**: `embed.FS` からファイルを自動スキャンし、プロンプトモードを自動マッピング。
* **🛠 Template-based Builder**: `text/template` を内包し、構造体データを注入して動的にプロンプトを生成。
* **🛡 Collision Detection**: モード名の衝突や空ファイルを初期化時に検知する堅牢なバリデーション。

### 📡 [mdcast] ドキュメント配信エンジン

* **📑 Markdown to HTML**: AI のレスポンス（Markdown）を、スタイル済みの完全な HTML ドキュメントへ変換。
* **🎨 Style-Injected Rendering**: 組み込みの CSS やテンプレートを使用して、即座に「見栄えの良い」成果物を出力。
* **🧩 Modular Architecture**: Converter, Renderer, Runner が分離されており、特定のロジックのみの差し替えが可能。

-----

## 🏗 プロジェクトレイアウト (Project Layout)

機能ごとに独立したモジュール構成を採用しており、必要な機能だけを選択して利用可能です。

```text
go-prompt-kit/
├── prompts/           # 【INPUT】プロンプト構築
│   └── builder.go     #   - モード管理とテンプレート実行
├── mdcast/            # 【OUTPUT】ドキュメント配信 (Cast)
│   ├── ports/         #   - 抽象インターフェース定義
│   ├── converter/     #   - Markdown 解析・タイトル抽出
│   ├── renderer/      #   - HTML レンダリング (CSS/Template)
│   ├── runner/        #   - 変換ワークフローの実行
│   └── builder/       #   - 具象インスタンスの構築
└── resource/          # 【BASE】共通基盤
    └── loader.go      #   - fs.FS からのアセット自動スキャン
```

-----

## 🤝 主な依存関係 (Dependencies)

* `text/template`: Go 標準のテンプレートエンジン
* `io/fs`: 抽象化されたファイルシステムインターフェース
* `embed`: 静的アセットのバイナリ埋め込み

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
