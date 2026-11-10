package prompts

import (
	"maps"
	"text/template"

	"github.com/shouni/go-prompt-kit/resource"
)

// config holds the settings for building a Builder.
type config struct {
	funcs            template.FuncMap
	partialPrefix    string
	defaultMode      string
	resourceOptions  []resource.Option
	splitFrontMatter bool
	trimPartials     bool

	// loadOnly holds the names of the given options that are only valid while loading. They are
	// recorded so that NewBuilder reports them as an error instead of silently ignoring them.
	loadOnly []string
}

// newConfig applies the defaults and then the options.
func newConfig(opts ...Option) *config {
	cfg := &config{
		partialPrefix: DefaultPartialPrefix,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Option is a function that changes the settings for building a Builder.
//
// WithPrefix / WithRecursive / WithExtensions / WithFrontMatter only mean something while reading
// files, so they are for LoadFS only; passing them to NewBuilder gives ErrLoadOnlyOption. The
// others work with both.
type Option func(*config)

// WithFuncs registers custom functions callable from templates. Several calls are merged, and a
// later function with the same name wins.
//
//	builder, err := prompts.NewBuilder(templates, prompts.WithFuncs(template.FuncMap{
//	    "join": strings.Join,
//	}))
func WithFuncs(funcs template.FuncMap) Option {
	return func(c *config) {
		if len(funcs) == 0 {
			return
		}
		if c.funcs == nil {
			c.funcs = make(template.FuncMap, len(funcs))
		}
		maps.Copy(c.funcs, funcs)
	}
}

// WithTrimPartials removes a partial's trailing newline before registering it.
//
// Files normally end with a newline, so a partial inserted mid-body adds a blank line at that point
// only. Referenced at the end of a body it makes no difference in the output, so it surfaces only
// where a partial is used mid-body. In formats where a blank line means a paragraph break (such as
// Markdown), the text is split into a separate paragraph right there.
//
// Only the trailing newlines (LF / CRLF) are removed. Trailing spaces and newlines inside the body
// stay, so an intended blank line is kept by writing one in the partial.
//
//	builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithTrimPartials())
//
// It does not trim by default so that the output of existing callers that reference partials at the
// end does not change.
func WithTrimPartials() Option {
	return func(c *config) {
		c.trimPartials = true
	}
}

// WithPartialPrefix changes the entry-name prefix that marks a partial (default
// DefaultPartialPrefix = "_"). The empty string disables partial detection, so every entry is
// exposed as a mode.
func WithPartialPrefix(prefix string) Option {
	return func(c *config) {
		c.partialPrefix = prefix
	}
}

// WithDefaultMode sets the fallback for an unregistered mode passed to Build. The empty string also
// falls back to the default mode, so callers need not check for an unspecified mode. If the given
// mode does not exist, NewBuilder returns ErrUnknownMode.
func WithDefaultMode(mode string) Option {
	return func(c *config) {
		if mode != "" {
			c.defaultMode = mode
		}
	}
}

// WithPrefix limits loading to files with the given prefix and removes the prefix from the mode
// name, like resource.WithPrefix. LoadFS only.
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithPrefix(prefix))
		c.loadOnly = append(c.loadOnly, "WithPrefix")
	}
}

// WithRecursive reads subdirectories recursively and makes the mode name the path relative to
// rootDir (e.g. "en/rock"), like resource.WithRecursive. LoadFS only.
func WithRecursive() Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithRecursive())
		c.loadOnly = append(c.loadOnly, "WithRecursive")
	}
}

// WithExtensions limits loading to files with the given extensions, like resource.WithExtensions.
// LoadFS only.
func WithExtensions(exts ...string) Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithExtensions(exts...))
		c.loadOnly = append(c.loadOnly, "WithExtensions")
	}
}

// WithFrontMatter splits the front matter at the top of each file off its body and registers only
// the body as a template. What was split off is available from Builder.FrontMatter /
// Builder.FrontMatters.
//
// Without it, front matter is treated as part of the body, and the metadata ends up at the top of
// the instruction sent to the AI. LoadFS only.
func WithFrontMatter() Option {
	return func(c *config) {
		c.splitFrontMatter = true
		c.loadOnly = append(c.loadOnly, "WithFrontMatter")
	}
}
