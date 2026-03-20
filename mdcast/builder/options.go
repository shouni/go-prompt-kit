package builder

// Option は Builder の設定を適用する関数型です。
type Option func(*Builder)

// WithEnableUnsafeHTML は、unsafe な HTML の有効/無効を設定します。
func WithEnableUnsafeHTML(enable bool) Option {
	return func(c *Builder) {
		c.config.enableUnsafeHTML = enable
	}
}

// WithEnableHardWraps は、ハードラップの有効/無効を設定します。
func WithEnableHardWraps(enable bool) Option {
	return func(c *Builder) {
		c.config.enableHardWraps = enable
	}
}

// WithHTMLMode は、Builder のモードを HTML に設定します。
func WithHTMLMode() Option {
	return func(c *Builder) {
		c.config.mode = htmlMode // builder.go 内の定数を使用
	}
}
