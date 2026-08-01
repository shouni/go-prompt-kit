// Package runner は、入力の変換とレンダリングを統合し、
// 完全なHTMLドキュメントを生成するランナーを提供します。
// 入力形式は注入する Converter が決めるため、Markdown 専用ではありません。
package runner

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/shouni/go-prompt-kit/md/ports"
)

const (
	defaultTitle = "Document"
	defaultLang  = "ja-jp"
)

// DocumentRunner は ports.Runner を実装する具体的な構造体です。
// Converter がMarkdownを扱うかJSONを扱うかには関知しません。
type DocumentRunner struct {
	converter    ports.Converter
	renderer     ports.Renderer
	lang         string
	defaultTitle string
}

// MarkdownToHTMLRunner は DocumentRunner の旧名です。
//
// Deprecated: 入力形式はConverter側で決まるため、DocumentRunner を使用してください。
type MarkdownToHTMLRunner = DocumentRunner

// NewDocumentRunner は新しいRunnerを初期化し、依存関係を注入します。
func NewDocumentRunner(converter ports.Converter, renderer ports.Renderer, opts ...Option) *DocumentRunner {
	cfg := &config{
		lang:         defaultLang,
		defaultTitle: defaultTitle,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &DocumentRunner{
		converter:    converter,
		renderer:     renderer,
		lang:         cfg.lang,
		defaultTitle: cfg.defaultTitle,
	}
}

// NewMarkdownToHTMLRunner は NewDocumentRunner の旧名です。
//
// Deprecated: NewDocumentRunner を使用してください。
func NewMarkdownToHTMLRunner(converter ports.Converter, renderer ports.Renderer, opts ...Option) *DocumentRunner {
	return NewDocumentRunner(converter, renderer, opts...)
}

// Run は、入力をHTMLフラグメントに変換し、
// タイトルと言語を付与して完全なHTMLドキュメントとしてレンダリングします。
// タイトルは title 引数 → Converter による入力からの抽出 → 既定値 の順に解決されます。
func (r *DocumentRunner) Run(title string, input []byte) (*bytes.Buffer, error) {
	// 1. 入力チェック
	if len(input) == 0 {
		return &bytes.Buffer{}, nil
	}

	// 2. 入力をHTMLフラグメントに変換
	htmlFragment, err := r.converter.Convert(input)
	if err != nil {
		return nil, fmt.Errorf("HTMLフラグメント生成エラー: %w", err)
	}

	// 3. HTMLドキュメントのレンダリング
	var htmlBuffer bytes.Buffer
	err = r.renderer.Render(&htmlBuffer, htmlFragment, r.lang, r.resolveTitle(title, input))
	if err != nil {
		slog.Error("HTMLレンダリングエラー。", "error", err)
		return nil, fmt.Errorf("HTMLレンダリングに失敗しました: %w", err)
	}

	return &htmlBuffer, nil
}

// resolveTitle は、呼び出し側の指定・入力からの抽出・既定値の順にタイトルを決定します。
func (r *DocumentRunner) resolveTitle(title string, input []byte) string {
	// 呼び出し側でタイトルが指定されている場合はそれを使用
	if title != "" {
		return title
	}

	// 指定がない場合は入力コンテンツから抽出
	if extracted := r.converter.ExtractTitleFromMarkdown(input); extracted != "" {
		return extracted
	}

	return r.defaultTitle
}
