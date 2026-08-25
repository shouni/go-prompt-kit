package renderer

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// Renderer は、HTMLフラグメントを完全なHTMLドキュメントへ組み立てる実装です。
type Renderer struct {
	tpl *template.Template
	css template.CSS
}

// New は Renderer を構築します。
// テンプレートとスタイルシートは構築時に一度だけ読み込み、以降の Render では読み直しません。
// オプションを指定しない場合は、埋め込みの template.html と default.css を使用します。
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
			return nil, fmt.Errorf("HTMLテンプレートのパースエラー: %w", err)
		}
		tpl = parsed
	}

	base := ""
	if cfg.css != nil {
		base = string(*cfg.css)
	} else {
		cssBytes, err := assets.ReadFile("default.css")
		if err != nil {
			return nil, fmt.Errorf("CSSファイルの読み込みエラー: %w", err)
		}
		base = string(cssBytes)
	}

	return &Renderer{
		tpl: tpl,
		css: template.CSS(joinCSS(base, cfg.extraCSS)),
	}, nil
}

// joinCSS は土台のスタイルシートへ追加分を連結します。
// 後ろにあるほど優先されるため、追加分は土台の指定を上書きできます。
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

// Render は、HTMLフラグメントをテンプレートへ埋め込み、完全なHTMLドキュメントとして
// writer へ書き出します。
//
// bodyHTML と構築時のスタイルシートはエスケープせずに埋め込むため、
// 信頼できる Converter の出力だけを渡してください。
// テンプレートの実行が途中で失敗した場合、writer にはそこまでの出力が残ります。
func (r *Renderer) Render(writer io.Writer, bodyHTML []byte, lang, title string) error {
	data := TemplateData{
		Lang:    lang,
		Title:   title,
		Style:   r.css,
		Content: template.HTML(bodyHTML),
	}

	if err := r.tpl.Execute(writer, data); err != nil {
		return fmt.Errorf("HTMLテンプレートの実行エラー: %w", err)
	}

	return nil
}
