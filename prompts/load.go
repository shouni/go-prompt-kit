package prompts

import (
	"io/fs"

	"github.com/shouni/go-prompt-kit/frontmatter"
	"github.com/shouni/go-prompt-kit/resource"
)

// LoadFS は、ファイルシステムからテンプレートを読み込んで Builder を構築します。
// embed.FS からの読み込みと Builder の構築をまとめた入口で、読み込みと構築の
// 間に手を入れる必要がなければ、resource.Load と NewBuilder を個別に呼ぶ代わりに
// これを使えます。
//
//	//go:embed prompts/*.md
//	var promptFiles embed.FS
//
//	builder, err := prompts.LoadFS(promptFiles, "prompts")
//
// プロンプトが先頭に front matter を持つ場合は WithFrontMatter を指定します。
// 本文だけがテンプレートとして登録され、front matter は Builder.FrontMatter から取得できます。
//
//	builder, err := prompts.LoadFS(promptFiles, "prompts", prompts.WithFrontMatter())
//	meta, err := frontmatter.DecodeMap[ModeInfo](builder.FrontMatters(), yaml.Unmarshal)
func LoadFS(fileSystem fs.FS, rootDir string, opts ...Option) (*Builder, error) {
	cfg := newConfig(opts...)

	templates, err := resource.Load(fileSystem, rootDir, cfg.resourceOptions...)
	if err != nil {
		return nil, err
	}

	var fronts map[string]string
	if cfg.splitFrontMatter {
		templates, fronts = frontmatter.SplitMap(templates)
	}

	// 読み込み専用オプションはここで消費済みのため、NewBuilder の検査は通しません。
	builder, err := newBuilder(templates, cfg)
	if err != nil {
		return nil, err
	}
	builder.fronts = fronts

	return builder, nil
}
