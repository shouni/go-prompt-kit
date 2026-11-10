package renderer

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// Renderer is the implementation that assembles an HTML fragment into a complete HTML document.
type Renderer struct {
	tpl *template.Template
	css template.CSS
}

// New builds a Renderer. The template and the stylesheet are read once at construction and never
// re-read by Render. Without options it uses the embedded template.html and default.css.
func New(opts ...Option) (*Renderer, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.err != nil {
		return nil, cfg.err
	}

	tpl := cfg.tpl
	if tpl == nil {
		parsed, err := template.ParseFS(assets, "template.html")
		if err != nil {
			return nil, fmt.Errorf("parse HTML template: %w", err)
		}
		tpl = parsed
	}

	base := ""
	if cfg.css != nil {
		base = string(*cfg.css)
	} else {
		cssBytes, err := assets.ReadFile("default.css")
		if err != nil {
			return nil, fmt.Errorf("read CSS file: %w", err)
		}
		base = string(cssBytes)
	}

	return &Renderer{
		tpl: tpl,
		css: template.CSS(joinCSS(base, cfg.extraCSS)),
	}, nil
}

// joinCSS appends the additions to the base stylesheet. Later rules take precedence, so the
// additions can override the base.
func joinCSS(base string, extra []string) string {
	parts := make([]string, 0, 1+len(extra))
	if base != "" {
		parts = append(parts, base)
	}
	for _, css := range extra {
		if css != "" {
			parts = append(parts, css)
		}
	}

	return strings.Join(parts, "\n")
}

// Render embeds the HTML fragment in the template and writes it to writer as a complete HTML
// document.
//
// bodyHTML and the stylesheet given at construction are embedded without escaping, so pass only the
// output of a trusted Converter. If template execution fails midway, the output up to that point
// remains in writer.
func (r *Renderer) Render(writer io.Writer, bodyHTML []byte, lang, title string) error {
	data := TemplateData{
		Lang:    lang,
		Title:   title,
		Style:   r.css,
		Content: template.HTML(bodyHTML),
	}

	if err := r.tpl.Execute(writer, data); err != nil {
		return fmt.Errorf("execute HTML template: %w", err)
	}

	return nil
}
