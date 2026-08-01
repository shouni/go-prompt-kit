// Package builder は、Converter/Renderer/Runnerを組み立てて
// ports.Runner を構築するファクトリを提供します。
package builder

import (
	"github.com/shouni/go-prompt-kit/md/converter"
	"github.com/shouni/go-prompt-kit/md/ports"
	"github.com/shouni/go-prompt-kit/md/renderer"
	"github.com/shouni/go-prompt-kit/md/runner"
)

// Option は Builder の設定を適用する関数型です。
type Option func(*Builder)

// WithEnableUnsafeHTML は、unsafe な HTML の有効/無効を設定します。
// WithConverter で Converter を明示注入した場合は無視されます。
func WithEnableUnsafeHTML(enable bool) Option {
	return func(c *Builder) {
		c.config.enableUnsafeHTML = enable
	}
}

// WithEnableHardWraps は、ハードラップの有効/無効を設定します。
// WithConverter で Converter を明示注入した場合は無視されます。
func WithEnableHardWraps(enable bool) Option {
	return func(c *Builder) {
		c.config.enableHardWraps = enable
	}
}

// WithHTMLMode は、Builder のモードを HTML に設定します。
//
// Deprecated: HTML が既定モードのため指定は不要です。
// 別のモードを指定する場合は WithMode を使用してください。
func WithHTMLMode() Option {
	return WithMode(htmlMode)
}

// WithMode は、BuildRunner が生成する Runner の種別を指定します。
// 現在サポートされるのは "html" のみで、それ以外を指定した場合
// BuildRunner が ErrUnsupportedMode を返します。
func WithMode(mode string) Option {
	return func(c *Builder) {
		c.config.mode = mode
	}
}

// WithConverter は、任意の ports.Converter を注入します。
// jsonconverter など、Markdown以外の入力を扱うパイプラインを組む場合に使用します。
// 指定した場合、Markdown用のGoldmarkConverterは構築されません。
func WithConverter(c ports.Converter) Option {
	return func(b *Builder) {
		b.converter = c
	}
}

// WithRenderer は、任意の ports.Renderer を注入します。
// 指定した場合、埋め込みアセットを使う既定のRendererは構築されません。
func WithRenderer(r ports.Renderer) Option {
	return func(b *Builder) {
		b.renderer = r
	}
}

// WithConverterOptions は、既定のGoldmarkConverterへ渡すオプションを追加します。
// WithConverter で Converter を明示注入した場合は無視されます。
func WithConverterOptions(opts ...converter.Option) Option {
	return func(b *Builder) {
		b.config.converterOptions = append(b.config.converterOptions, opts...)
	}
}

// WithRendererOptions は、既定のRendererへ渡すオプションを追加します。
// テンプレートやCSSを差し替える場合に使用します。
// WithRenderer で Renderer を明示注入した場合は無視されます。
func WithRendererOptions(opts ...renderer.Option) Option {
	return func(b *Builder) {
		b.config.rendererOptions = append(b.config.rendererOptions, opts...)
	}
}

// WithLang は、生成するHTMLドキュメントの lang 属性を設定します（既定は "ja-jp"）。
func WithLang(lang string) Option {
	return func(b *Builder) {
		b.config.runnerOptions = append(b.config.runnerOptions, runner.WithLang(lang))
	}
}

// WithDefaultTitle は、タイトルを解決できなかった場合の既定タイトルを設定します。
func WithDefaultTitle(title string) Option {
	return func(b *Builder) {
		b.config.runnerOptions = append(b.config.runnerOptions, runner.WithDefaultTitle(title))
	}
}
