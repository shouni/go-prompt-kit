package prompts

import (
	"io/fs"

	"github.com/shouni/go-prompt-kit/frontmatter"
	"github.com/shouni/go-prompt-kit/resource"
)

// LoadFS reads templates from a file system and builds a Builder. It is the entry point combining
// loading from an embed.FS with building the Builder; when nothing needs to happen between loading
// and building, use it instead of calling resource.Load and NewBuilder separately.
//
//	//go:embed prompts/*.md
//	var promptFiles embed.FS
//
//	builder, err := prompts.LoadFS(promptFiles, "prompts")
//
// When the prompts start with front matter, pass WithFrontMatter. Only the body is registered as a
// template, and the front matter is available from Builder.FrontMatter.
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

	// The load-only options were consumed here, so skip NewBuilder's check.
	builder, err := newBuilder(templates, cfg)
	if err != nil {
		return nil, err
	}
	builder.fronts = fronts

	return builder, nil
}
