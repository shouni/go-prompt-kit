// Package htmldoc provides a pipeline that assembles an AI response (Markdown or JSON) into a
// complete HTML document.
//
// Conversion runs in two stages, Converter (input → HTML fragment) and Renderer (fragment →
// complete HTML), and Document ties the two together. By default a Markdown Converter and a
// Renderer using the embedded assets are built, so two lines are enough:
//
//	doc, err := htmldoc.New()
//	err = doc.Run(w, "", input)
//
// To use another Converter, for example when the input is JSON, inject it with WithConverter.
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

// Document implements ports.Runner; it ties a Converter and a Renderer together and writes a
// complete HTML document. It does not care whether the Converter handles Markdown or JSON.
type Document struct {
	converter    ports.Converter
	renderer     ports.Renderer
	lang         string
	defaultTitle string
}

// New builds a Document. A Converter / Renderer can be injected with WithConverter / WithRenderer,
// and the default implementations (the Markdown Converter, the Renderer with embedded assets) are
// built only when nothing is injected. It returns an error when building the default Renderer
// fails.
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
			return nil, fmt.Errorf("initialize renderer: %w", err)
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

// Run converts the input into an HTML fragment, adds the title and language, and writes it to
// writer as a complete HTML document. The title resolves in this order: the title argument →
// extraction from the input by the Converter → the default. An empty input writes nothing and
// returns nil.
func (d *Document) Run(writer io.Writer, title string, input []byte) error {
	if len(input) == 0 {
		return nil
	}

	htmlFragment, resolved, err := d.convert(input, title)
	if err != nil {
		return err
	}

	if err := d.renderer.Render(writer, htmlFragment, d.lang, resolved); err != nil {
		return fmt.Errorf("render HTML: %w", err)
	}

	return nil
}

// convert returns the HTML fragment and the resolved title.
//
// When no title is given and the Converter implements ports.TitledConverter, conversion and
// extraction are requested together. Calling Convert and ExtractTitle separately would parse the
// same input twice.
func (d *Document) convert(input []byte, title string) (fragment []byte, resolved string, err error) {
	titled, ok := d.converter.(ports.TitledConverter)
	if title != "" || !ok {
		fragment, err = d.converter.Convert(input)
		if err != nil {
			return nil, "", fmt.Errorf("generate HTML fragment: %w", err)
		}

		return fragment, d.resolveTitle(title, input), nil
	}

	fragment, extracted, err := titled.ConvertWithTitle(input)
	if err != nil {
		return nil, "", fmt.Errorf("generate HTML fragment: %w", err)
	}
	if extracted == "" {
		extracted = d.defaultTitle
	}

	return fragment, extracted, nil
}

// resolveTitle decides the title in this order: the caller's title, extraction from the input, the
// default. It is the path for Converters that do not implement ports.TitledConverter (when they do,
// convert resolves both at once and this is not reached).
func (d *Document) resolveTitle(title string, input []byte) string {
	if title != "" {
		return title
	}

	if extracted := d.converter.ExtractTitle(input); extracted != "" {
		return extracted
	}

	return d.defaultTitle
}
