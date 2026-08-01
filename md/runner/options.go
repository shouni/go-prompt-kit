package runner

// config は DocumentRunner の構築設定を保持します。
type config struct {
	lang         string
	defaultTitle string
}

// Option は DocumentRunner の設定オプションを定義する関数型です。
type Option func(*config)

// WithLang は、生成するHTMLドキュメントの lang 属性を設定します（既定は "ja-jp"）。
// 空文字を渡した場合は既定値が維持されます。
func WithLang(lang string) Option {
	return func(c *config) {
		if lang != "" {
			c.lang = lang
		}
	}
}

// WithDefaultTitle は、タイトルが指定されず入力からも抽出できなかった場合に
// 使用するタイトルを設定します（既定は "Document"）。
// 空文字を渡した場合は既定値が維持されます。
func WithDefaultTitle(title string) Option {
	return func(c *config) {
		if title != "" {
			c.defaultTitle = title
		}
	}
}
