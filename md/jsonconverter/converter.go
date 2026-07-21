// Package jsonconverter は、任意の構造を持つJSON入力を、呼び出し側が指定した
// html/template を通じてHTMLフラグメントへ変換する汎用コンバーターを提供します。
// GoldmarkConverter がMarkdownの中身に関知しないのと同様、JSONのスキーマや
// テンプレートの中身には関知しません — それらは呼び出し側の責務です。
package jsonconverter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/shouni/go-prompt-kit/md/ports"
)

var _ ports.Converter = (*JSONConverter)(nil)

const defaultTitleKey = "title"

// JSONConverter は md/ports.Converter を実装し、JSON入力を呼び出し側の
// テンプレートでHTMLフラグメントへ変換します。
type JSONConverter struct {
	tpl      *template.Template
	titleKey string
}

// New は、HTMLフラグメントの生成に使うテンプレートを受け取り JSONConverter を構築します。
func New(tpl *template.Template, opts ...Option) *JSONConverter {
	c := &JSONConverter{
		tpl:      tpl,
		titleKey: defaultTitleKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert は、JSON入力をデコードし、テンプレートに渡してHTMLフラグメントを生成します。
func (c *JSONConverter) Convert(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil
	}

	var data any
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, fmt.Errorf("JSON入力のパースに失敗しました: %w", err)
	}

	var buf bytes.Buffer
	if err := c.tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("HTMLフラグメントのレンダリングに失敗しました: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractTitleFromMarkdown は、JSON入力のトップレベルオブジェクトから
// titleKey(デフォルト "title")に対応する文字列値を抽出します。
// メソッド名は md/ports.Converter インターフェースに合わせています(Markdown専用ではありません)。
func (c *JSONConverter) ExtractTitleFromMarkdown(input []byte) string {
	var data map[string]any
	if err := json.Unmarshal(input, &data); err != nil {
		return ""
	}

	if v, ok := data[c.titleKey].(string); ok {
		return v
	}
	return ""
}
