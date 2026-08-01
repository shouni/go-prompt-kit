package prompts

import (
	"maps"
	"text/template"

	"github.com/shouni/go-prompt-kit/resource"
)

// config は Builder の構築設定を保持します。
type config struct {
	funcs           template.FuncMap
	partialPrefix   string
	defaultMode     string
	resourceOptions []resource.Option
}

// newConfig は既定値を適用したうえでオプションを反映します。
func newConfig(opts ...Option) *config {
	cfg := &config{
		partialPrefix: DefaultPartialPrefix,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Option は Builder の構築設定を変更する関数型です。
// NewBuilder と LoadFS の両方に渡せますが、読み込みに関するオプション
// （WithRecursive / WithExtensions）が意味を持つのは LoadFS のみです。
type Option func(*config)

// WithFuncs は、テンプレートから呼び出せるカスタム関数を登録します。
// 複数回指定した場合はマージされ、同名の関数は後勝ちになります。
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

// WithPartialPrefix は、partial とみなすエントリ名の接頭辞を変更します
// （既定は DefaultPartialPrefix = "_"）。
// 空文字を指定すると partial の判定を行わず、全エントリがモードとして公開されます。
func WithPartialPrefix(prefix string) Option {
	return func(c *config) {
		c.partialPrefix = prefix
	}
}

// WithDefaultMode は、未登録のモードが Build に渡された場合のフォールバック先を設定します。
// 空文字が渡された場合も既定モードへ委ねられるため、
// 呼び出し側でモード未指定を判定する必要がなくなります。
// 指定したモードが実在しない場合、NewBuilder は ErrUnknownMode を返します。
func WithDefaultMode(mode string) Option {
	return func(c *config) {
		if mode != "" {
			c.defaultMode = mode
		}
	}
}

// WithRecursive は、サブディレクトリを再帰的に読み込みます。
// モード名は rootDir からの相対パスになります（例: "en/rock"）。
// LoadFS でのみ意味を持ちます。
func WithRecursive() Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithRecursive())
	}
}

// WithExtensions は、読み込む対象を指定した拡張子のファイルだけに限定します。
// LoadFS でのみ意味を持ちます。
func WithExtensions(exts ...string) Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithExtensions(exts...))
	}
}
