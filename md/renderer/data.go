package renderer

import (
	"embed"
	"html/template"
)

//go:embed *.html *.css
var assets embed.FS

// TemplateData は、汎用Markdown用のテンプレートに渡す値を保持します。
type TemplateData struct {
	Lang    string
	Title   string
	Style   template.CSS
	Content template.HTML
}
