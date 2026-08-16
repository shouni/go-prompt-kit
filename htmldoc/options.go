package htmldoc

import (
	"github.com/shouni/go-prompt-kit/htmldoc/markdown"
	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/shouni/go-prompt-kit/htmldoc/renderer"
)

// config は Document の構築設定を保持します。
type config struct {
	converter        ports.Converter
	renderer         ports.Renderer
	converterOptions []markdown.Option
	rendererOptions  []renderer.Option
	lang             string
	defaultTitle     string
}

// Option は Document の設定オプションを定義する関数型です。
type Option func(*config)

// WithConverter は、任意の ports.Converter を注入します。
// jsondoc など、Markdown以外の入力を扱うパイプラインを組む場合に使用します。
// 指定した場合、既定の markdown.Converter は構築されず WithConverterOptions は無視されます。
func WithConverter(c ports.Converter) Option {
	return func(cfg *config) {
		cfg.converter = c
	}
}

// WithRenderer は、任意の ports.Renderer を注入します。
// 指定した場合、埋め込みアセットを使う既定のRendererは構築されず
// WithRendererOptions は無視されます。
func WithRenderer(r ports.Renderer) Option {
	return func(cfg *config) {
		cfg.renderer = r
	}
}

// WithConverterOptions は、既定の markdown.Converter へ渡すオプションを追加します。
// WithConverter で Converter を明示注入した場合は無視されます。
func WithConverterOptions(opts ...markdown.Option) Option {
	return func(cfg *config) {
		cfg.converterOptions = append(cfg.converterOptions, opts...)
	}
}

// WithRendererOptions は、既定のRendererへ渡すオプションを追加します。
// テンプレートやCSSを差し替える場合に使用します。
// WithRenderer で Renderer を明示注入した場合は無視されます。
func WithRendererOptions(opts ...renderer.Option) Option {
	return func(cfg *config) {
		cfg.rendererOptions = append(cfg.rendererOptions, opts...)
	}
}

// WithLang は、生成するHTMLドキュメントの lang 属性を設定します（既定は "ja-jp"）。
// 空文字を渡した場合は既定値が維持されます。
func WithLang(lang string) Option {
	return func(cfg *config) {
		if lang != "" {
			cfg.lang = lang
		}
	}
}

// WithDefaultTitle は、タイトルが指定されず入力からも抽出できなかった場合に
// 使用するタイトルを設定します（既定は "Document"）。
// 空文字を渡した場合は既定値が維持されます。
func WithDefaultTitle(title string) Option {
	return func(cfg *config) {
		if title != "" {
			cfg.defaultTitle = title
		}
	}
}
