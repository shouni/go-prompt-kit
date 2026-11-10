// Package ports defines the interfaces (ports) HTML document generation depends on. Keeping them in
// a dedicated package, apart from the implementations, lets both the implementations (markdown /
// jsondoc / renderer) and the consumer (htmldoc) reference the interfaces without import cycles.
package ports

import (
	"io"
)

// Runner abstracts the core service that writes input as a complete HTML document. The injected
// Converter decides the input format, so this abstraction is format-agnostic.
type Runner interface {
	Run(writer io.Writer, title string, input []byte) error
}

// Converter is the interface of a service that converts input into an HTML fragment (the body part,
// not a whole document).
type Converter interface {
	// Convert converts the input into an HTML fragment.
	Convert(input []byte) ([]byte, error)
	// ExtractTitle extracts the document title from the input. It returns the empty string when
	// there is none.
	ExtractTitle(input []byte) string
}

// TitledConverter is a Converter that can convert and extract the title in a single parse.
//
// Document works with a plain Converter, but then every automatic title extraction makes Convert
// and ExtractTitle each parse the same input, so the input is parsed twice. When this is
// implemented, Document calls it instead and parses once.
//
// The bundled markdown.Converter and jsondoc.Converter both implement it.
type TitledConverter interface {
	Converter

	// ConvertWithTitle parses the input once and returns the HTML fragment and the title. When no
	// title can be extracted, title is empty (not an error).
	//
	// The returned values must match those of calling Convert and ExtractTitle separately.
	ConvertWithTitle(input []byte) (fragment []byte, title string, err error)
}

// Renderer is the interface that assembles an HTML fragment into a complete HTML document.
type Renderer interface {
	Render(writer io.Writer, bodyHTML []byte, lang, title string) error
}
