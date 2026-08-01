package prompts

import (
	"io/fs"

	"github.com/shouni/go-prompt-kit/resource"
)

// LoadFS は、ファイルシステムからテンプレートを読み込んで Builder を構築します。
// embed.FS からの読み込みと Builder の構築をまとめた入口で、
// resource.Load と NewBuilder を個別に呼ぶ場合と等価です。
//
//	//go:embed prompts/prompt_*.md
//	var files embed.FS
//
//	builder, err := prompts.LoadFS(files, "prompts", "prompt_")
func LoadFS(fileSystem fs.FS, rootDir, prefix string, opts ...Option) (*Builder, error) {
	cfg := newConfig(opts...)

	templates, err := resource.Load(fileSystem, rootDir, prefix, cfg.resourceOptions...)
	if err != nil {
		return nil, err
	}

	return NewBuilder(templates, opts...)
}
