// Package renderer provides a ports.Renderer implementation that assembles an HTML fragment into a
// complete HTML document.
//
// It assembles with html/template; the template and the stylesheet default to the embedded
// template.html / default.css and can be replaced by options.
package renderer

import (
	"embed"
	"html/template"
)

//go:embed *.html *.css
var assets embed.FS

// TemplateData holds the values passed to the document template.
//
// A template supplied through WithTemplate / WithTemplateText can reference only these four. They
// have the same shape as the default template (template.html):
//
//	<html lang="{{.Lang}}">
//	<title>{{.Title}}</title>
//	<style>{{.Style}}</style>
//	{{.Content}}
//
// Style and Content are template.CSS / template.HTML so that html/template does not escape them.
// CSS is whatever WithCSS gave, and Content is the fragment the Converter built, inserted as is.
// Both are assumed to be trusted input; this is not a place to feed user input directly.
type TemplateData struct {
	// Lang is the language tag in <html lang> (htmldoc.WithLang, default "ja-jp").
	Lang string
	// Title is the string in <title>. When empty, the Converter picks it from the body.
	Title string
	// Style is the content of <style>. It is not escaped.
	Style template.CSS
	// Content is the HTML fragment of the body. It is not escaped.
	Content template.HTML
}
