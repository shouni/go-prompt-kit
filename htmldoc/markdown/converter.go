// Package markdown provides a ports.Converter implementation that converts Markdown into an HTML
// fragment.
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

// Converter is the Markdown ports.Converter implementation, built on goldmark.
type Converter struct {
	md goldmark.Markdown
}

// New creates a Markdown Converter.
func New(opts ...Option) *Converter {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	// GFM is always on; the extensions given as options are added to it.
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

// Convert converts Markdown into HTML.
func (c *Converter) Convert(input []byte) ([]byte, error) {
	fragment, _, err := c.convert(input, false)

	return fragment, err
}

// ConvertWithTitle parses the input once and returns the HTML fragment and the title (the
// ports.TitledConverter implementation). The result is the same as calling Convert and ExtractTitle
// in turn, but the syntax tree is built only once.
func (c *Converter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	return c.convert(input, true)
}

// convert does the work of building the syntax tree, rendering and extracting the title.
// goldmark.Markdown.Convert only "parses and renders", so here the two stages are called separately
// to keep the parse result at hand.
func (c *Converter) convert(input []byte, withTitle bool) ([]byte, string, error) {
	doc := c.md.Parser().Parse(text.NewReader(input))

	var buf bytes.Buffer
	if err := c.md.Renderer().Render(&buf, input, doc); err != nil {
		return nil, "", fmt.Errorf("convert markdown to HTML: %w", err)
	}

	var title string
	if withTitle {
		title = firstHeadingText(doc, input)
	}

	return buf.Bytes(), title, nil
}

// ExtractTitle extracts the text of the first heading as the title. It searches the syntax tree
// built by goldmark's parser, so lines starting with "#" inside code blocks, closing "#" sequences
// and setext headings are all handled correctly. It returns the empty string when there is no
// heading.
//
// When the title is needed along with the conversion, use ConvertWithTitle, which does not parse
// the input twice.
func (c *Converter) ExtractTitle(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	return firstHeadingText(c.md.Parser().Parse(text.NewReader(input)), input)
}

// firstHeadingText returns the text of the first heading in the syntax tree.
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
		slog.Warn("error while extracting the Markdown title", "error", err)
		return ""
	}

	return title
}

// inlineText recursively collects plain text from the inline elements under a node. Decorations
// such as emphasis and links are removed, leaving only their text.
func inlineText(n ast.Node, source []byte) string {
	var sb strings.Builder

	var walk func(ast.Node)
	walk = func(node ast.Node) {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			switch v := child.(type) {
			case *ast.Text:
				sb.Write(v.Segment.Value(source))
				// Add a space so that words do not run together when a heading spans several lines.
				if v.SoftLineBreak() || v.HardLineBreak() {
					sb.WriteByte(' ')
				}
			case *ast.String:
				sb.Write(v.Value)
			case *ast.AutoLink:
				sb.Write(v.URL(source))
			default:
				// Emphasis / Link / CodeSpan and the like: follow the children's text.
				walk(child)
			}
		}
	}
	walk(n)

	return sb.String()
}
