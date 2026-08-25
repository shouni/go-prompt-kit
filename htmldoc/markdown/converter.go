// Package markdown は、MarkdownをHTMLフラグメントへ変換する ports.Converter 実装を提供します。
package markdown

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var (
	_ ports.Converter       = (*Converter)(nil)
	_ ports.TitledConverter = (*Converter)(nil)
)

// Converter は goldmark を用いた Markdown 用の ports.Converter 実装です。
type Converter struct {
	md goldmark.Markdown
}

// New は Markdown 用の Converter を作成します。
func New(opts ...Option) *Converter {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	// GFM は常に有効にし、オプションで指定された拡張を追加します。
	extensions := append([]goldmark.Extender{extension.GFM}, cfg.extensions...)

	goldmarkOptions := []goldmark.Option{
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(cfg.parserOptions...),
		goldmark.WithRendererOptions(cfg.rendererOptions...),
	}
	goldmarkOptions = append(goldmarkOptions, cfg.goldmarkOptions...)

	return &Converter{
		md: goldmark.New(goldmarkOptions...),
	}
}

// Convert は Markdown を HTML に変換します。
func (c *Converter) Convert(input []byte) ([]byte, error) {
	fragment, _, err := c.convert(input, false)

	return fragment, err
}

// ConvertWithTitle は、入力を1回だけ解析してHTMLフラグメントとタイトルを返します
// （ports.TitledConverter の実装です）。
// Convert と ExtractTitle を続けて呼ぶのと結果は同じですが、構文木の構築は1回で済みます。
func (c *Converter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	return c.convert(input, true)
}

// convert は、構文木の構築・レンダリング・タイトル抽出をまとめた実体です。
// goldmark.Markdown.Convert は「解析してレンダリングする」だけなので、
// 解析結果を手元に残せるようここでは2段階に分けて呼びます。
func (c *Converter) convert(input []byte, withTitle bool) ([]byte, string, error) {
	doc := c.md.Parser().Parse(text.NewReader(input))

	var buf bytes.Buffer
	if err := c.md.Renderer().Render(&buf, input, doc); err != nil {
		return nil, "", fmt.Errorf("markdownからHTMLへの変換に失敗しました: %w", err)
	}

	var title string
	if withTitle {
		title = firstHeadingText(doc, input)
	}

	return buf.Bytes(), title, nil
}

// ExtractTitle は、最初の見出しのテキストをタイトルとして抽出します。
// goldmark のパーサーで構文木を構築してから探索するため、コードブロック内の
// "#" で始まる行や、閉じの "#" 列、setext形式の見出しも正しく扱えます。
// 見出しが存在しない場合は空文字を返します。
//
// 変換と同時にタイトルが必要な場合は、入力を2度解析しない ConvertWithTitle を使ってください。
func (c *Converter) ExtractTitle(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	return firstHeadingText(c.md.Parser().Parse(text.NewReader(input)), input)
}

// firstHeadingText は、構文木の中で最初に現れる見出しのテキストを返します。
func firstHeadingText(doc ast.Node, source []byte) string {
	var title string
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		title = strings.TrimSpace(inlineText(heading, source))
		return ast.WalkStop, nil
	})
	if err != nil {
		slog.Warn("Markdownタイトル抽出中にエラーが発生しました", "error", err)
		return ""
	}

	return title
}

// inlineText は、ノード配下のインライン要素からプレーンテキストを再帰的に収集します。
// 強調やリンクなどの装飾記法は取り除かれ、その中身のテキストだけが残ります。
func inlineText(n ast.Node, source []byte) string {
	var sb strings.Builder

	var walk func(ast.Node)
	walk = func(node ast.Node) {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			switch v := child.(type) {
			case *ast.Text:
				sb.Write(v.Segment.Value(source))
				// 見出しが複数行にまたがる場合に単語が繋がらないよう空白を補います。
				if v.SoftLineBreak() || v.HardLineBreak() {
					sb.WriteByte(' ')
				}
			case *ast.String:
				sb.Write(v.Value)
			case *ast.AutoLink:
				sb.Write(v.URL(source))
			default:
				// Emphasis / Link / CodeSpan などは子のテキストを辿ります。
				walk(child)
			}
		}
	}
	walk(n)

	return sb.String()
}
