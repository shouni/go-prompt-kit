package builder

import (
	"errors"
	"fmt"

	"github.com/shouni/go-prompt-kit/md/converter"
	"github.com/shouni/go-prompt-kit/md/ports"
	"github.com/shouni/go-prompt-kit/md/renderer"
	"github.com/shouni/go-prompt-kit/md/runner"
)

const htmlMode = "html"

// ErrUnsupportedMode は、BuildRunner が未サポートのモードを指定された場合に返されます。
var ErrUnsupportedMode = errors.New("未サポートのモード")

// config は Builder の設定を保持します。
type config struct {
	enableUnsafeHTML bool
	enableHardWraps  bool
	mode             string
	converterOptions []converter.Option
	rendererOptions  []renderer.Option
	runnerOptions    []runner.Option
}

// Builder は依存関係を管理し、適切なRunnerを生成します。
type Builder struct {
	config    config
	converter ports.Converter
	renderer  ports.Renderer
}

// New はコンポーネントを初期化して Builder を作成します。
// Converter / Renderer は WithConverter / WithRenderer で注入でき、
// 注入がない場合のみ Markdown 用の既定実装が構築されます。
func New(options ...Option) (*Builder, error) {
	// 1. デフォルト設定の適用
	builder := &Builder{
		config: config{
			enableUnsafeHTML: false,
			enableHardWraps:  false,
			mode:             htmlMode,
		},
	}

	// 2. オプションによる設定の上書き
	for _, opt := range options {
		opt(builder)
	}

	// 3. Converterの構築（明示注入がない場合のみ）
	if builder.converter == nil {
		opts := append([]converter.Option{
			converter.WithUnsafeHTML(builder.config.enableUnsafeHTML),
			converter.WithHardWraps(builder.config.enableHardWraps),
		}, builder.config.converterOptions...)
		builder.converter = converter.NewGoldmarkConverter(opts...)
	}

	// 4. Rendererの構築（明示注入がない場合のみ）
	if builder.renderer == nil {
		r, err := renderer.NewRenderer(builder.config.rendererOptions...)
		if err != nil {
			return nil, fmt.Errorf("rendererの初期化エラー: %w", err)
		}
		builder.renderer = r
	}

	return builder, nil
}

// BuildRunner は設定されたモードに応じて最適な Runner を返します。
func (b *Builder) BuildRunner() (ports.Runner, error) {
	switch b.config.mode {
	case htmlMode, "":
		return runner.NewDocumentRunner(b.converter, b.renderer, b.config.runnerOptions...), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMode, b.config.mode)
	}
}
