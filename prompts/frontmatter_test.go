package prompts

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/frontmatter"
)

// frontMatterFS is a file system holding prompts with front matter. The library does not decide the
// front matter format, so the tests put JSON there, which the standard library alone can read.
func frontMatterFS() fstest.MapFS {
	return fstest.MapFS{
		"prompts/review.md": &fstest.MapFile{Data: []byte(
			"---\n{\"direction\":\"レビュー\"}\n---\n差分を確認してください。\n{{template \"_output\" .}}\n")},
		"prompts/summary.md": &fstest.MapFile{Data: []byte(
			"---\n{\"direction\":\"要約\"}\n---\n要約してください。\n")},
		"prompts/plain.md": &fstest.MapFile{Data: []byte(
			"front matter のないプロンプトです。\n")},
		"prompts/_output.md": &fstest.MapFile{Data: []byte(
			"---\n{\"direction\":\"部品\"}\n---\nJSONで出力してください。\n")},
	}
}

func TestLoadFS_WithFrontMatter(t *testing.T) {
	b, err := LoadFS(frontMatterFS(), "prompts", WithFrontMatter())
	require.NoError(t, err)

	t.Run("本文だけがテンプレートとして登録される", func(t *testing.T) {
		got, err := b.Build("summary", nil)
		require.NoError(t, err)

		assert.Equal(t, "要約してください。\n", got)
		assert.NotContains(t, got, "direction", "front matter remained in the body")
	})

	t.Run("partial の参照も本文だけが展開される", func(t *testing.T) {
		got, err := b.Build("review", nil)
		require.NoError(t, err)

		assert.Equal(t, "差分を確認してください。\nJSONで出力してください。\n\n", got)
	})

	t.Run("切り離した front matter を取り出せる", func(t *testing.T) {
		assert.Equal(t, `{"direction":"レビュー"}`, b.FrontMatter("review"))
		assert.Equal(t, `{"direction":"要約"}`, b.FrontMatter("summary"))
	})

	t.Run("partial の front matter も保持する", func(t *testing.T) {
		assert.Equal(t, `{"direction":"部品"}`, b.FrontMatter("_output"))
	})

	t.Run("front matter を持たないエントリと未知のエントリは空文字", func(t *testing.T) {
		assert.Empty(t, b.FrontMatter("plain"))
		assert.Empty(t, b.FrontMatter("存在しないモード"))
	})

	t.Run("FrontMatters は decode へそのまま渡せる", func(t *testing.T) {
		type modeInfo struct {
			Direction string `json:"direction"`
		}

		infos, err := frontmatter.DecodeMap[modeInfo](b.FrontMatters(), json.Unmarshal)
		require.NoError(t, err)

		assert.Equal(t, "レビュー", infos["review"].Direction)
		assert.Zero(t, infos["plain"], "an entry without front matter gets the zero value")
	})

	t.Run("FrontMatters の書き換えは Builder に影響しない", func(t *testing.T) {
		got := b.FrontMatters()
		got["review"] = "書き換え"
		delete(got, "summary")

		assert.Equal(t, `{"direction":"レビュー"}`, b.FrontMatter("review"))
		assert.Equal(t, `{"direction":"要約"}`, b.FrontMatter("summary"))
	})
}

// TestLoadFS_WithoutFrontMatter confirms that without the option the front matter stays in the body
// (splitting is an explicit choice).
func TestLoadFS_WithoutFrontMatter(t *testing.T) {
	b, err := LoadFS(frontMatterFS(), "prompts")
	require.NoError(t, err)

	got, err := b.Build("summary", nil)
	require.NoError(t, err)

	assert.Contains(t, got, "direction")
	assert.Empty(t, b.FrontMatter("summary"), "nothing was split off, so nothing is kept")
	assert.Empty(t, b.FrontMatters())
}
