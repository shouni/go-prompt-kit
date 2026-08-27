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
//
// WithTemplate / WithTemplateText で差し替えるテンプレートは、この 4 つだけを
// 参照できます。既定のテンプレート（template.html）と同じ形です。
//
//	<html lang="{{.Lang}}">
//	<title>{{.Title}}</title>
//	<style>{{.Style}}</style>
//	{{.Content}}
//
// Style と Content が template.CSS / template.HTML なのは、html/template に
// エスケープさせないためです。CSS は WithCSS で与えたものが、Content は
// Converter が組んだフラグメントがそのまま入ります。どちらも信頼できる入力で
// あることが前提で、利用者の入力を直接流し込む場所ではありません。
type TemplateData struct {
	// Lang は <html lang> に入る言語タグです（htmldoc.WithLang、既定 "ja-jp"）。
	Lang string
	// Title は <title> に入る文字列です。空の場合は Converter が本文から拾います。
	Title string
	// Style は <style> の中身です。エスケープされません。
	Style template.CSS
	// Content は本文の HTML フラグメントです。エスケープされません。
	Content template.HTML
}
