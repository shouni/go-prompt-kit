package prompts

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFS(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/prompt_review.md":  {Data: []byte("レビュー: {{.Target}}")},
		"prompts/prompt_release.md": {Data: []byte("リリース手順")},
		"prompts/README.txt":        {Data: []byte("無視される")},
	}

	t.Run("読み込みとBuilder構築をまとめて行える", func(t *testing.T) {
		b, err := LoadFS(mockFS, "prompts", WithPrefix("prompt_"))
		require.NoError(t, err)

		assert.Equal(t, []string{"release", "review"}, b.Modes())

		got, err := b.Build("review", struct{ Target string }{Target: "差分"})
		require.NoError(t, err)
		assert.Equal(t, "レビュー: 差分", got)
	})

	t.Run("存在しないディレクトリはエラーを伝播する", func(t *testing.T) {
		_, err := LoadFS(mockFS, "missing", WithPrefix("prompt_"))
		require.Error(t, err)
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("該当ファイルが無い場合はErrEmptyTemplates", func(t *testing.T) {
		_, err := LoadFS(mockFS, "prompts", WithPrefix("存在しない接頭辞_"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyTemplates)
	})
}

func TestLoadFS_Recursive(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/ja/_output.md": {Data: []byte("出力: JSON")},
		"prompts/ja/rock.md":    {Data: []byte("ロック\n{{template \"ja/_output\" .}}")},
		"prompts/en/_output.md": {Data: []byte("Output: JSON")},
		"prompts/en/rock.md":    {Data: []byte("Rock\n{{template \"en/_output\" .}}")},
	}

	b, err := LoadFS(mockFS, "prompts", WithRecursive())
	require.NoError(t, err)

	t.Run("言語ディレクトリがモード名の接頭辞になる", func(t *testing.T) {
		assert.Equal(t, []string{"en/rock", "ja/rock"}, b.Modes())
	})

	t.Run("言語ごとのpartialが正しく解決される", func(t *testing.T) {
		gotJA, err := b.Build("ja/rock", nil)
		require.NoError(t, err)
		assert.Equal(t, "ロック\n出力: JSON", gotJA)

		gotEN, err := b.Build("en/rock", nil)
		require.NoError(t, err)
		assert.Equal(t, "Rock\nOutput: JSON", gotEN)
	})
}

func TestLoadFS_WithExtensions(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/a.md":   {Data: []byte("markdown")},
		"prompts/b.tmpl": {Data: []byte("template")},
		"prompts/c.txt":  {Data: []byte("text")},
	}

	b, err := LoadFS(mockFS, "prompts", WithExtensions(".md", "tmpl"))
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b"}, b.Modes())
}
