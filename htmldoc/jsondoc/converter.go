// Package jsondoc provides a ports.Converter implementation that converts JSON input of any
// structure into an HTML fragment through an html/template given by the caller. Just as
// markdown.Converter does not care about the content of the Markdown, this package does not care
// about the JSON schema or the template's content — those are the caller's responsibility.
//
// Numbers are passed to the template as json.Number (a string exactly as written in the input).
// Going through float64 would put large integers into exponent notation and round decimals ending
// in 0, so the numbers in the document would disagree with the input.
package jsondoc

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
)

var (
	_ ports.Converter       = (*Converter)(nil)
	_ ports.TitledConverter = (*Converter)(nil)
)

const defaultTitleKey = "title"

// Converter is the JSON ports.Converter implementation; it converts JSON input into an HTML
// fragment with the caller's template.
type Converter struct {
	tpl      *template.Template
	titleKey string
}

// New builds a Converter from the template used to generate the HTML fragment.
func New(tpl *template.Template, opts ...Option) *Converter {
	c := &Converter{
		tpl:      tpl,
		titleKey: defaultTitleKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert decodes the JSON input and passes it to the template to generate an HTML fragment.
func (c *Converter) Convert(input []byte) ([]byte, error) {
	fragment, _, err := c.convert(input, false)

	return fragment, err
}

// ConvertWithTitle decodes the input once and returns the HTML fragment and the title (the
// ports.TitledConverter implementation). The result is the same as calling Convert and ExtractTitle
// in turn, but the input is decoded only once.
func (c *Converter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	return c.convert(input, true)
}

// convert does the work of decoding, rendering and getting the title.
func (c *Converter) convert(input []byte, withTitle bool) ([]byte, string, error) {
	if len(input) == 0 {
		return nil, "", nil
	}

	data, err := decode(input)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := c.tpl.Execute(&buf, data); err != nil {
		return nil, "", fmt.Errorf("render HTML fragment: %w", err)
	}

	var title string
	if withTitle {
		title = c.titleOf(data)
	}

	return buf.Bytes(), title, nil
}

// decode reads the JSON input into an any.
//
// Numbers are kept as json.Number (a string exactly as written). Converting them to the default
// float64 would turn a large integer into exponent notation such as "1.234567890123e+12" and round
// a decimal ending in 0 (0.30) to 0.3. Templates output that as is, so the number displayed would
// disagree with the number in the input. Writing documents is the purpose of this package, so the
// literal text wins.
//
// To compare or compute with numbers in a template, register a function that takes a json.Number
// and converts it, via Funcs.
func decode(input []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()

	var data any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("parse JSON input: %w", err)
	}

	// Unlike json.Unmarshal, a Decoder skips whatever follows the value, so confirm that the input
	// ends after one value.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse JSON input: unexpected data after the value")
	}

	return data, nil
}

// titleOf takes the string value of titleKey from the top level of the decoded value.
func (c *Converter) titleOf(data any) string {
	object, ok := data.(map[string]any)
	if !ok {
		return ""
	}

	title, _ := object[c.titleKey].(string)

	return title
}

// ExtractTitle extracts the string value for titleKey (default "title") from the top-level object
// of the JSON input.
//
// It does not decode the whole input and stops reading as soon as the key is found. A title
// therefore comes back even when the input is broken after the key (Convert reports an error for
// that input anyway). When the title is needed along with the conversion, use ConvertWithTitle,
// which does not read the input twice.
func (c *Converter) ExtractTitle(input []byte) string {
	dec := jsontext.NewDecoder(bytes.NewReader(input))
	if dec.PeekKind() != jsontext.KindBeginObject {
		return ""
	}
	if _, err := dec.ReadToken(); err != nil {
		return ""
	}

	for dec.PeekKind() != jsontext.KindEndObject {
		name, err := dec.ReadToken()
		if err != nil {
			return ""
		}

		if name.String() != c.titleKey {
			if err := dec.SkipValue(); err != nil {
				return ""
			}
			continue
		}

		// A value that is not a string is not treated as a title.
		if dec.PeekKind() != jsontext.KindString {
			return ""
		}
		value, err := dec.ReadToken()
		if err != nil {
			return ""
		}

		return value.String()
	}

	return ""
}
