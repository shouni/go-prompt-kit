package htmldoc

import (
	"github.com/shouni/go-prompt-kit/htmldoc/markdown"
	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/shouni/go-prompt-kit/htmldoc/renderer"
)

// config holds the settings for building a Document.
type config struct {
	converter        ports.Converter
	renderer         ports.Renderer
	converterOptions []markdown.Option
	rendererOptions  []renderer.Option
	lang             string
	defaultTitle     string
}

// Option is a function that configures a Document.
type Option func(*config)

// WithConverter injects any ports.Converter. Use it to build a pipeline for input other than
// Markdown, such as jsondoc. When set, the default markdown.Converter is not built and
// WithConverterOptions is ignored.
func WithConverter(c ports.Converter) Option {
	return func(cfg *config) {
		cfg.converter = c
	}
}

// WithRenderer injects any ports.Renderer. When set, the default Renderer using the embedded assets
// is not built and WithRendererOptions is ignored.
func WithRenderer(r ports.Renderer) Option {
	return func(cfg *config) {
		cfg.renderer = r
	}
}

// WithConverterOptions adds options passed to the default markdown.Converter. Ignored when a
// Converter is injected explicitly with WithConverter.
func WithConverterOptions(opts ...markdown.Option) Option {
	return func(cfg *config) {
		cfg.converterOptions = append(cfg.converterOptions, opts...)
	}
}

// WithRendererOptions adds options passed to the default Renderer. Use it to replace the template
// or the CSS. Ignored when a Renderer is injected explicitly with WithRenderer.
func WithRendererOptions(opts ...renderer.Option) Option {
	return func(cfg *config) {
		cfg.rendererOptions = append(cfg.rendererOptions, opts...)
	}
}

// WithLang sets the lang attribute of the generated HTML document (default "ja-jp"). The empty
// string keeps the default.
func WithLang(lang string) Option {
	return func(cfg *config) {
		if lang != "" {
			cfg.lang = lang
		}
	}
}

// WithDefaultTitle sets the title used when none is given and none can be extracted from the input
// (default "Document"). The empty string keeps the default.
func WithDefaultTitle(title string) Option {
	return func(cfg *config) {
		if title != "" {
			cfg.defaultTitle = title
		}
	}
}
