// Package renderer は、HTMLフラグメントを完全なHTMLドキュメントへ組み立てる
// ports.Renderer 実装を提供します。
//
// 組み立てには html/template を使い、テンプレートとスタイルシートは
// 埋め込みの template.html / default.css を既定として、オプションで差し替えられます。
package renderer

import (
	"embed"
	"html/template"
)

//go:embed *.html *.css
var assets embed.FS

// TemplateData は、ドキュメントテンプレートへ渡す値を保持します。
type TemplateData struct {
	Lang    string
	Title   string
	Style   template.CSS
	Content template.HTML
}
