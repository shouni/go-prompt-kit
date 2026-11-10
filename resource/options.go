package resource

import "strings"

// config holds the settings for Load.
type config struct {
	recursive  bool
	extensions []string
	prefix     string
}

// newConfig returns the settings with the options applied.
func newConfig(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Option is a function that changes how Load behaves.
type Option func(*config)

// WithPrefix limits loading to files with the given prefix. The prefix is removed from the mode
// name ("prompt_review.md" with prefix "prompt_" becomes "review"). Without it every file is
// loaded, and the mode name is the file name without its extension.
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.prefix = prefix
	}
}

// WithRecursive walks subdirectories recursively. The mode name is then the path relative to
// rootDir without its extension (e.g. "prompts/en/rock.md" with rootDir "prompts" gives "en/rock").
func WithRecursive() Option {
	return func(c *config) {
		c.recursive = true
	}
}

// WithExtensions limits loading to files with the given extensions. The leading dot is optional
// (".md" and "md" are the same). Without it every extension is loaded.
func WithExtensions(exts ...string) Option {
	return func(c *config) {
		for _, ext := range exts {
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			c.extensions = append(c.extensions, ext)
		}
	}
}

// matchExtension reports whether a file's extension is one of the targets.
func (c *config) matchExtension(ext string) bool {
	if len(c.extensions) == 0 {
		return true
	}
	for _, want := range c.extensions {
		if strings.EqualFold(ext, want) {
			return true
		}
	}
	return false
}
