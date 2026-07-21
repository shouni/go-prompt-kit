package jsonconverter

// Option は JSONConverter の設定オプションを定義する関数型です。
type Option func(*JSONConverter)

// WithTitleKey は、ExtractTitleFromMarkdown がタイトルとして参照する
// トップレベルJSONキーを変更します(デフォルトは "title")。
func WithTitleKey(key string) Option {
	return func(c *JSONConverter) {
		if key != "" {
			c.titleKey = key
		}
	}
}
