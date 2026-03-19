package renderer

import (
	"fmt"
	"html/template"
	"io"
)

// Renderer は、汎用的なMarkdownからHTMLを生成する実装です。
type Renderer struct {
	tpl *template.Template
	css template.CSS // CSSをキャッシュしてパフォーマンスを向上させます
}

// NewRenderer は、アセットを事前にロードしてインスタンスを生成します。
func NewRenderer() (*Renderer, error) {
	// テンプレートのパース
	tpl, err := template.ParseFS(assets, "template.html")
	if err != nil {
		return nil, fmt.Errorf("HTMLテンプレートのパースエラー: %w", err)
	}

	// CSSの事前読み込み（キャッシュ化）
	cssBytes, err := assets.ReadFile("default.css")
	if err != nil {
		return nil, fmt.Errorf("CSSファイルの読み込みエラー: %w", err)
	}

	return &Renderer{
		tpl: tpl,
		css: template.CSS(cssBytes),
	}, nil
}

// Render は Renderer 用の実装です。
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
