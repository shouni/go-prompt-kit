package builder

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/md/converter"
	"github.com/shouni/go-prompt-kit/md/ports"
	"github.com/shouni/go-prompt-kit/md/renderer"
	"github.com/shouni/go-prompt-kit/md/runner"
)

const htmlMode = "html"

// config は Builder の設定を保持します。
type config struct {
	enableUnsafeHTML bool
	enableHardWraps  bool
	mode             string
}

// Builder は依存関係を管理し、適切なRunnerを生成します。
type Builder struct {
	config    config
	converter ports.Converter
	renderer  ports.Renderer
}

// New はコンポーネントを初期化して Builder を作成します。
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

	// 3. Converterの構築
	opts := []converter.Option{
		converter.WithUnsafeHTML(builder.config.enableUnsafeHTML),
		converter.WithHardWraps(builder.config.enableHardWraps),
	}
	c := converter.NewGoldmarkConverter(opts...)

	// 4. Rendererの構築
	r, err := renderer.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("Rendererの初期化エラー: %w", err)
	}

	builder.converter = c
	builder.renderer = r

	return builder, nil
}

// BuildRunner は設定されたモードに応じて最適な Runner を返します。
func (b *Builder) BuildRunner() (ports.Runner, error) {
	switch b.config.mode {
	case htmlMode, "":
		return runner.NewMarkdownToHtmlRunner(b.converter, b.renderer), nil
	default:
		return nil, fmt.Errorf("未サポートのモード: %s", b.config.mode)
	}
}
