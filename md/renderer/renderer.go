package renderer

import (
	"fmt"
	"html/template"
	"io"
)

// Renderer は、HTMLフラグメントを完全なHTMLドキュメントへ組み立てる実装です。
type Renderer struct {
	tpl *template.Template
	css template.CSS // CSSをキャッシュしてパフォーマンスを向上させます
}

// NewRenderer は、アセットを事前にロードしてインスタンスを生成します。
// オプションを指定しない場合は、埋め込みの template.html と default.css を使用します。
func NewRenderer(opts ...Option) (*Renderer, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.err != nil {
		return nil, cfg.err
	}

	// テンプレートのパース（未指定なら埋め込みアセット）
	tpl := cfg.tpl
	if tpl == nil {
		parsed, err := template.ParseFS(assets, "template.html")
		if err != nil {
			return nil, fmt.Errorf("HTMLテンプレートのパースエラー: %w", err)
		}
		tpl = parsed
	}

	// CSSの事前読み込み（キャッシュ化）
	var css template.CSS
	if cfg.css != nil {
		css = *cfg.css
	} else {
		cssBytes, err := assets.ReadFile("default.css")
		if err != nil {
			return nil, fmt.Errorf("CSSファイルの読み込みエラー: %w", err)
		}
		css = template.CSS(cssBytes)
	}

	return &Renderer{
		tpl: tpl,
		css: css,
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
