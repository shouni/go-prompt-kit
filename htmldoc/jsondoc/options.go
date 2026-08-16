package jsondoc

// Option は Converter の設定オプションを定義する関数型です。
type Option func(*Converter)

// WithTitleKey は、ExtractTitle がタイトルとして参照する
// トップレベルJSONキーを変更します(デフォルトは "title")。
func WithTitleKey(key string) Option {
	return func(c *Converter) {
		if key != "" {
			c.titleKey = key
		}
	}
}
