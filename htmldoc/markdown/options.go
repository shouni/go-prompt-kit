package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
)

// config は Converter の構築設定を保持します。
// goldmark.New へ渡すオプションを種別ごとに蓄積し、New の中で一度だけ適用します。
type config struct {
	rendererOptions []renderer.Option
	parserOptions   []parser.Option
	extensions      []goldmark.Extender
	goldmarkOptions []goldmark.Option
}

// Option は Converter の設定オプションを定義する関数型です。
type Option func(*config)

// WithUnsafeHTML は、Goldmarkレンダラーで「安全でない」HTML出力を許可するオプションです。
// 有効にすると Markdown 中の生HTMLがそのまま出力されるため、
// 信頼できない入力に対しては使用しないでください。
func WithUnsafeHTML(enable bool) Option {
	return func(c *config) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithUnsafe())
		}
	}
}

// WithHardWraps は、Markdown内の単純な改行を <br> タグに変換するオプションです。
func WithHardWraps(enable bool) Option {
	return func(c *config) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithHardWraps())
		}
	}
}

// WithAutoHeadingID は、見出しへ自動的に id 属性を付与するオプションです。
// 目次からアンカーリンクを張る場合に有効にします。
func WithAutoHeadingID(enable bool) Option {
	return func(c *config) {
		if enable {
			c.parserOptions = append(c.parserOptions, parser.WithAutoHeadingID())
		}
	}
}

// WithTypographer は、引用符やダッシュなどを活字風の記号へ置き換えるオプションです。
func WithTypographer(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.Typographer)
		}
	}
}

// WithCJK は、日本語などの東アジア言語向けの改行処理を有効にします。
//
// Markdown では段落の途中の改行（ソフト改行）が出力HTMLにもそのまま残り、
// ブラウザ上では空白として描画されます。単語を空白で区切る言語では正しい挙動ですが、
// 日本語では文の途中に空白が入ります。
//
//	これは日本語の文章で、
//	途中で改行しています。
//
//	無効: <p>これは日本語の文章で、\n途中で改行しています。</p>  → 「、 途中で」と表示される
//	有効: <p>これは日本語の文章で、途中で改行しています。</p>
//
// 取り除かれるのは改行の前後がともに東アジアの全角文字である場合だけなので、
// 日本語と英文が混在する文書でも英文側の区切りは保たれます。
//
// htmldoc の既定の lang は "ja-jp" ですが、他の拡張オプションと揃えて既定では無効です。
func WithCJK(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.CJK)
		}
	}
}

// WithFootnotes は、脚注記法（[^1]）を有効にするオプションです。
func WithFootnotes(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.Footnote)
		}
	}
}

// WithGoldmarkOptions は、上記のオプションで表現できない設定を
// goldmark.Option として直接渡すための拡張点です。
func WithGoldmarkOptions(opts ...goldmark.Option) Option {
	return func(c *config) {
		c.goldmarkOptions = append(c.goldmarkOptions, opts...)
	}
}
