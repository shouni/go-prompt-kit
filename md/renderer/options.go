package renderer

import (
	"fmt"
	"html/template"
)

// config は Renderer の構築設定を保持します。
// tpl / css が nil のままなら、埋め込みアセット（template.html / default.css）が使われます。
type config struct {
	tpl      *template.Template
	css      *template.CSS
	extraCSS []string
	err      error
}

// Option は Renderer の設定オプションを定義する関数型です。
type Option func(*config)

// WithTemplate は、パース済みのテンプレートを使用します。
// テンプレートは TemplateData（Lang / Title / Style / Content）を受け取れる必要があります。
func WithTemplate(tpl *template.Template) Option {
	return func(c *config) {
		if tpl == nil {
			c.err = fmt.Errorf("テンプレートがnilです")
			return
		}
		c.tpl = tpl
	}
}

// WithTemplateText は、テンプレート文字列をパースして使用します。
// パースに失敗した場合は NewRenderer がエラーを返します。
func WithTemplateText(text string) Option {
	return func(c *config) {
		tpl, err := template.New("template.html").Parse(text)
		if err != nil {
			c.err = fmt.Errorf("HTMLテンプレートのパースエラー: %w", err)
			return
		}
		c.tpl = tpl
	}
}

// WithCSS は、埋め込みの default.css に代えて任意のスタイルシートを使用します。
// テンプレートの <style> へそのまま挿入されるため、信頼できる内容のみを渡してください。
// 空文字を渡すとスタイルなしでレンダリングされます。
func WithCSS(css string) Option {
	return func(c *config) {
		style := template.CSS(css)
		c.css = &style
	}
}

// WithExtraCSS は、土台となるスタイルシートの後ろへ追加のスタイルを連結します。
// 既定の見た目を保ったまま独自の部品スタイルだけを足したい場合に使います。
// 土台は WithCSS を指定していればその内容、指定がなければ埋め込みの default.css です。
//
// 後ろへ足されるため、同じセレクタを書けば土台の指定を上書きできます。
// 複数回指定した場合は指定順に連結されます。
func WithExtraCSS(css string) Option {
	return func(c *config) {
		if css == "" {
			return
		}
		c.extraCSS = append(c.extraCSS, css)
	}
}
