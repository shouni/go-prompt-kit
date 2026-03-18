# ✍️ Go Prompt Kit

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/go-prompt-kit)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/go-prompt-kit)](https://github.com/shouni/go-prompt-kit/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - プロンプト管理を、もっと「動的」で「型安全」に。

**Go Prompt Kit** は、AI（Gemini 等）へのプロンプトテンプレートを Go プログラム内で柔軟に管理するためのツールキットです。

`embed.FS` を利用したリソースの自動スキャン機能と、`text/template` をベースとした強力なビルド機能を組み合わせることで、プロンプトの追加・変更に伴うボイラープレートコードを排除し、AI 連携ロジックの保守性を最大化します。

---

## ✨ 提供機能 (Features)

* **📦 Dynamic Resource Loader**: `embed.FS` から指定したディレクトリ・接頭辞に一致するファイルを自動的にマッピングします。
* **🛠 Template-based Builder**: `text/template` を内包し、任意の構造体データを注入して動的にプロンプトを生成します。
* **🛡 Collision Detection**: モード名の衝突や空ファイルの読み込みを初期化時に検知する堅牢なバリデーション。
* **🧩 High Extensibility**: `fs.FS` インターフェースに依存しているため、テスト時のモック化やローカルファイル読み込みへの切り替えが容易。

---

## 🏗 プロジェクトレイアウト (Project Layout)

機能ごとにパッケージが独立しており、用途に合わせて柔軟に組み合わせて利用可能です。

```text
gcp-kit/
├── resource/          # リソースロード基盤
│   └── loader.go      # fs.FS からファイルを動的にスキャンし map[string]string を生成
└── prompts/           # プロンプトエンジン
    └── builder.go     # モード管理とテンプレート実行（text/template ラッパー）
```

-----

## 🤝 主な依存関係 (Dependencies)

* `text/template`: Go 標準のテンプレートエンジン
* `io/fs`: 抽象化されたファイルシステムインターフェース
* `embed`: 静的アセットのバイナリ埋め込み

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。

---

