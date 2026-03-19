package runner

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/shouni/go-prompt-kit/mdcast/ports"
)

// MarkdownToHtmlRunner は Runner インターフェースを実装する具体的な構造体です。
type MarkdownToHtmlRunner struct {
	converter ports.Converter
	renderer  ports.Renderer
}

// NewMarkdownToHtmlRunner は新しいRunnerを初期化し、依存関係を注入します。
func NewMarkdownToHtmlRunner(converter ports.Converter, renderer ports.Renderer) *MarkdownToHtmlRunner {
	return &MarkdownToHtmlRunner{
		converter: converter,
		renderer:  renderer,
	}
}

// Run は、MarkdownバイトスライスをHTMLフラグメントに変換し、
// 指定されたタイトルとロケールを使用して完全なHTMLドキュメントとしてレンダリングします。
// title 引数を string 型に戻します。
func (r *MarkdownToHtmlRunner) Run(title string, markdown []byte) (*bytes.Buffer, error) {
	const defaultTitle = "Document"
	const defaultLocale = "ja-jp" // ロケールはハードコード

	// 1. 入力チェック
	if len(markdown) == 0 {
		return &bytes.Buffer{}, nil
	}

	// 2. MarkdownをHTMLフラグメントに変換
	htmlFragment, err := r.converter.Convert(markdown)
	if err != nil {
		return nil, fmt.Errorf("HTMLフラグメント生成エラー: %w", err)
	}

	// 3. HTMLドキュメントのレンダリング
	var htmlBuffer bytes.Buffer
	var articleTitle string
	if title != "" {
		// コマンドラインでタイトルが指定されている場合はそれを使用
		articleTitle = title
	} else {
		// 指定がない場合はMarkdownコンテンツから抽出
		articleTitle = r.converter.ExtractTitleFromMarkdown(markdown)
		if articleTitle == "" {
			articleTitle = defaultTitle
		}
	}
	err = r.renderer.Render(&htmlBuffer, htmlFragment, defaultLocale, articleTitle)
	if err != nil {
		slog.Error("HTMLレンダリングエラー。", "error", err)
		return nil, fmt.Errorf("HTMLレンダリングに失敗しました: %w", err)
	}

	return &htmlBuffer, nil
}
