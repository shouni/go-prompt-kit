// Package jsondoc は、任意の構造を持つJSON入力を、呼び出し側が指定した
// html/template を通じてHTMLフラグメントへ変換する ports.Converter 実装を提供します。
// markdown.Converter がMarkdownの中身に関知しないのと同様、JSONのスキーマや
// テンプレートの中身には関知しません — それらは呼び出し側の責務です。
package jsondoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
)

var _ ports.Converter = (*Converter)(nil)

const defaultTitleKey = "title"

// Converter は JSON 用の ports.Converter 実装で、
// JSON入力を呼び出し側のテンプレートでHTMLフラグメントへ変換します。
type Converter struct {
	tpl      *template.Template
	titleKey string
}

// New は、HTMLフラグメントの生成に使うテンプレートを受け取り Converter を構築します。
func New(tpl *template.Template, opts ...Option) *Converter {
	c := &Converter{
		tpl:      tpl,
		titleKey: defaultTitleKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert は、JSON入力をデコードし、テンプレートに渡してHTMLフラグメントを生成します。
func (c *Converter) Convert(input []byte) ([]byte, error) {
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

// ExtractTitle は、JSON入力のトップレベルオブジェクトから
// titleKey(デフォルト "title")に対応する文字列値を抽出します。
func (c *Converter) ExtractTitle(input []byte) string {
	var data map[string]any
	if err := json.Unmarshal(input, &data); err != nil {
		return ""
	}

	if v, ok := data[c.titleKey].(string); ok {
		return v
	}
	return ""
}
