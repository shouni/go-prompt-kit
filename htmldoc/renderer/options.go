package renderer

import (
	"fmt"
	"html/template"
)

// config holds the settings for building a Renderer. While tpl / css stay nil, the embedded assets
// (template.html / default.css) are used.
type config struct {
	tpl      *template.Template
	css      *template.CSS
	extraCSS []string
	err      error
}

// Option is a function that configures a Renderer.
type Option func(*config)

// WithTemplate uses an already parsed template. The template must accept a TemplateData (Lang /
// Title / Style / Content).
func WithTemplate(tpl *template.Template) Option {
	return func(c *config) {
		if tpl == nil {
			c.err = fmt.Errorf("template is nil")
			return
		}
		c.tpl = tpl
	}
}

// WithTemplateText parses a template string and uses it. If parsing fails, New returns the error.
func WithTemplateText(text string) Option {
	return func(c *config) {
		tpl, err := template.New("template.html").Parse(text)
		if err != nil {
			c.err = fmt.Errorf("parse HTML template: %w", err)
			return
		}
		c.tpl = tpl
	}
}

// WithCSS uses any stylesheet instead of the embedded default.css. It is inserted into the
// template's <style> as is, so pass only trusted content. The empty string renders without styles.
func WithCSS(css string) Option {
	return func(c *config) {
		style := template.CSS(css)
		c.css = &style
	}
}

// WithExtraCSS appends additional styles after the base stylesheet. Use it to add only your own
// component styles while keeping the default look. The base is the WithCSS content when given,
// otherwise the embedded default.css.
//
// It is appended after the base, so writing the same selector overrides the base. Several calls are
// joined in the order given.
func WithExtraCSS(css string) Option {
	return func(c *config) {
		if css == "" {
			return
		}
		c.extraCSS = append(c.extraCSS, css)
	}
}
