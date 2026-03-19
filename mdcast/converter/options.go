package converter

import "github.com/yuin/goldmark/renderer/html"

// Option は GoldmarkConverter の設定オプションを定義する関数型なのだ。
type Option func(*GoldmarkConverter)

// WithUnsafeHTML は、Goldmarkレンダラーで「安全でない」HTML出力を許可するオプションなのだ。
func WithUnsafeHTML(enable bool) Option {
	return func(c *GoldmarkConverter) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithUnsafe())
		}
	}
}

// WithHardWraps は、Markdown内の単純な改行を <br> タグに変換するオプションなのだ。
func WithHardWraps(enable bool) Option {
	return func(c *GoldmarkConverter) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithHardWraps())
		}
	}
}
