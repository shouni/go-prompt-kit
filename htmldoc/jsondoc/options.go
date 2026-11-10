package jsondoc

// Option is a function that configures a Converter.
type Option func(*Converter)

// WithTitleKey changes the top-level JSON key ExtractTitle reads as the title (default "title").
func WithTitleKey(key string) Option {
	return func(c *Converter) {
		if key != "" {
			c.titleKey = key
		}
	}
}
