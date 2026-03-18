package resource

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/prompt_review.md":   {Data: []byte("Review Template")},
		"prompts/prompt_release.md":  {Data: []byte("Release Template")},
		"prompts/other_file.txt":     {Data: []byte("Ignore Me")},
		"other_dir/prompt_unused.md": {Data: []byte("Wrong Dir")},
	}

	t.Run("正常系: プロンプトが正しくロードされること", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", "prompt_")

		// Load自体の失敗は致命的なので require を使用
		require.NoError(t, err, "Load に失敗しました")

		// アサーションの統一
		assert.Len(t, templates, 2, "期待されるテンプレート数は 2 です")
		assert.Equal(t, "Review Template", templates["review"], "review の内容が不正です")
		assert.Equal(t, "Release Template", templates["release"], "release の内容が不正です")
	})

	t.Run("異常系: 存在しないディレクトリを指定した場合エラーになること", func(t *testing.T) {
		_, err := Load(mockFS, "non_existent", "prompt_")

		// エラーが発生することを検証
		assert.Error(t, err, "存在しないディレクトリ指定時はエラーが返るべきです")
	})
}
