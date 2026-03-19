package converter

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
)

// ATX形式の見出し（# Title）にマッチする正規表現です。
// 1個から6個の # で始まり、その後に1つ以上の空白、そしてタイトル文字列が続くパターンです。
var atxHeadingRegex = regexp.MustCompile(`^#{1,6}\s+(.+)`)

// GoldmarkConverter は goldmark ライブラリを使用した実体です。
type GoldmarkConverter struct {
	md              goldmark.Markdown
	rendererOptions []renderer.Option
}

// NewGoldmarkConverter は新しいインスタンスを作成します。
func NewGoldmarkConverter(opts ...Option) *GoldmarkConverter {
	c := &GoldmarkConverter{
		rendererOptions: []renderer.Option{},
	}

	for _, opt := range opts {
		opt(c)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
		goldmark.WithRendererOptions(c.rendererOptions...),
	)

	c.md = md
	c.rendererOptions = nil
	return c
}

// Convert は Markdown を HTML に変換します。
func (c *GoldmarkConverter) Convert(input []byte) ([]byte, error) {
	var buf bytes.Buffer

	if err := c.md.Convert(input, &buf); err != nil {
		return nil, fmt.Errorf("markdownからHTMLへの変換に失敗しました: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractTitleFromMarkdown は、正規表現を使って最初の見出しをスマートに抜き出す。
func (c *GoldmarkConverter) ExtractTitleFromMarkdown(input []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(input))

	for scanner.Scan() {
		// インデントされている可能性も考慮して TrimSpace
		line := strings.TrimSpace(scanner.Text())

		// 正規表現で見出し行をキャプチャする
		matches := atxHeadingRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			// matches[0] は行全体、matches[1] がカッコ内のタイトル部分なのだ
			return strings.TrimSpace(matches[1])
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("Markdownタイトル抽出中にエラーが発生しました", "error", err)
	}

	return ""
}
