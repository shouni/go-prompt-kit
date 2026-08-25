// Package htmldoc は、AIの応答（MarkdownやJSON）を完全なHTMLドキュメントへ
// 組み立てるパイプラインを提供します。
//
// 変換は Converter（入力 → HTMLフラグメント）とRenderer（フラグメント → 完全なHTML）の
// 2段階で行われ、Document がその2つを束ねます。既定ではMarkdown用のConverterと
// 埋め込みアセットを使うRendererが構築されるため、次の2行で利用できます。
//
//	doc, err := htmldoc.New()
//	err = doc.Run(w, "", input)
//
// 入力がJSONの場合など、別の Converter を使う場合は WithConverter で注入します。
//
//	doc, err := htmldoc.New(htmldoc.WithConverter(jsondoc.New(tpl)))
package htmldoc

import (
	"fmt"
	"io"

	"github.com/shouni/go-prompt-kit/htmldoc/markdown"
	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/shouni/go-prompt-kit/htmldoc/renderer"
)

const (
	defaultTitle = "Document"
	defaultLang  = "ja-jp"
)

var _ ports.Runner = (*Document)(nil)

// Document は ports.Runner の実装で、ConverterとRendererを束ねて
// 完全なHTMLドキュメントを書き出します。
// Converter がMarkdownを扱うかJSONを扱うかには関知しません。
type Document struct {
	converter    ports.Converter
	renderer     ports.Renderer
	lang         string
	defaultTitle string
}

// New は Document を構築します。
// Converter / Renderer は WithConverter / WithRenderer で注入でき、
// 注入がない場合のみ既定の実装（Markdown用Converter・埋め込みアセットのRenderer）が
// 構築されます。既定のRendererの構築に失敗した場合はエラーを返します。
func New(opts ...Option) (*Document, error) {
	cfg := &config{
		lang:         defaultLang,
		defaultTitle: defaultTitle,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	converter := cfg.converter
	if converter == nil {
		converter = markdown.New(cfg.converterOptions...)
	}

	rend := cfg.renderer
	if rend == nil {
		r, err := renderer.New(cfg.rendererOptions...)
		if err != nil {
			return nil, fmt.Errorf("rendererの初期化エラー: %w", err)
		}
		rend = r
	}

	return &Document{
		converter:    converter,
		renderer:     rend,
		lang:         cfg.lang,
		defaultTitle: cfg.defaultTitle,
	}, nil
}

// Run は、入力をHTMLフラグメントに変換し、タイトルと言語を付与して
// 完全なHTMLドキュメントとして writer へ書き出します。
// タイトルは title 引数 → Converter による入力からの抽出 → 既定値 の順に解決されます。
// 入力が空の場合は何も書き出さずに nil を返します。
func (d *Document) Run(writer io.Writer, title string, input []byte) error {
	if len(input) == 0 {
		return nil
	}

	htmlFragment, resolved, err := d.convert(input, title)
	if err != nil {
		return err
	}

	if err := d.renderer.Render(writer, htmlFragment, d.lang, resolved); err != nil {
		return fmt.Errorf("HTMLレンダリングに失敗しました: %w", err)
	}

	return nil
}

// convert は、HTMLフラグメントと確定したタイトルを返します。
//
// タイトルの指定がなく Converter が ports.TitledConverter を実装している場合は、
// 変換と抽出をまとめて依頼します。そうしないと Convert と ExtractTitle が
// 同じ入力をそれぞれ解析することになり、入力を2回解析します。
func (d *Document) convert(input []byte, title string) (fragment []byte, resolved string, err error) {
	titled, ok := d.converter.(ports.TitledConverter)
	if title != "" || !ok {
		fragment, err = d.converter.Convert(input)
		if err != nil {
			return nil, "", fmt.Errorf("HTMLフラグメント生成エラー: %w", err)
		}

		return fragment, d.resolveTitle(title, input), nil
	}

	fragment, extracted, err := titled.ConvertWithTitle(input)
	if err != nil {
		return nil, "", fmt.Errorf("HTMLフラグメント生成エラー: %w", err)
	}
	if extracted == "" {
		extracted = d.defaultTitle
	}

	return fragment, extracted, nil
}

// resolveTitle は、呼び出し側の指定・入力からの抽出・既定値の順にタイトルを決定します。
// 入力からの抽出が要らない場合に無駄な解析をしないよう、Converter が
// ports.TitledConverter を実装していない場合の経路でだけ使います。
func (d *Document) resolveTitle(title string, input []byte) string {
	// 呼び出し側でタイトルが指定されている場合はそれを使用します。
	if title != "" {
		return title
	}

	// 指定がない場合は入力コンテンツから抽出します。
	if extracted := d.converter.ExtractTitle(input); extracted != "" {
		return extracted
	}

	return d.defaultTitle
}
