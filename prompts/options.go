package prompts

import (
	"maps"
	"text/template"

	"github.com/shouni/go-prompt-kit/resource"
)

// config は Builder の構築設定を保持します。
type config struct {
	funcs            template.FuncMap
	partialPrefix    string
	defaultMode      string
	resourceOptions  []resource.Option
	splitFrontMatter bool

	// loadOnly は、指定された「読み込み時のみ有効なオプション」の名前です。
	// NewBuilder に渡された場合、黙って無視せずエラーにするために記録します。
	loadOnly []string
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
//
// WithPrefix / WithRecursive / WithExtensions / WithFrontMatter は
// ファイルを読むときにしか意味がないため LoadFS 専用で、
// NewBuilder に渡すと ErrLoadOnlyOption になります。それ以外は両方で使えます。
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

// WithPrefix は resource.WithPrefix と同じく、対象を指定した接頭辞を持つ
// ファイルへ限定し、接頭辞をモード名から取り除きます。LoadFS 専用です。
func WithPrefix(prefix string) Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithPrefix(prefix))
		c.loadOnly = append(c.loadOnly, "WithPrefix")
	}
}

// WithRecursive は resource.WithRecursive と同じく、サブディレクトリを再帰的に
// 読み込み、モード名を rootDir からの相対パスにします（例: "en/rock"）。LoadFS 専用です。
func WithRecursive() Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithRecursive())
		c.loadOnly = append(c.loadOnly, "WithRecursive")
	}
}

// WithExtensions は resource.WithExtensions と同じく、対象を指定した拡張子の
// ファイルへ限定します。LoadFS 専用です。
func WithExtensions(exts ...string) Option {
	return func(c *config) {
		c.resourceOptions = append(c.resourceOptions, resource.WithExtensions(exts...))
		c.loadOnly = append(c.loadOnly, "WithExtensions")
	}
}

// WithFrontMatter は、各ファイルの先頭にある front matter を本文から切り離し、
// 本文だけをテンプレートとして登録します。切り離した内容は
// Builder.FrontMatter / Builder.FrontMatters から取得できます。
//
// 指定しない場合、front matter は本文の一部として扱われ、
// AIへの指示の先頭にメタデータが紛れ込みます。LoadFS 専用です。
func WithFrontMatter() Option {
	return func(c *config) {
		c.splitFrontMatter = true
		c.loadOnly = append(c.loadOnly, "WithFrontMatter")
	}
}
