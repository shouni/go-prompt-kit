package resource

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_NonRecursiveByDefault(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/top.md":       {Data: []byte("トップ")},
		"prompts/ja/nested.md": {Data: []byte("ネスト")},
	}

	templates, err := Load(mockFS, "prompts")
	require.NoError(t, err)

	assert.Len(t, templates, 1, "既定ではサブディレクトリを走査しません")
	assert.Equal(t, "トップ", templates["top"])
}

func TestLoad_WithRecursive(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/top.md":             {Data: []byte("トップ")},
		"prompts/ja/nested.md":       {Data: []byte("ネスト")},
		"prompts/en/deep/deeper.md":  {Data: []byte("深い")},
		"prompts/ja/_partial.md":     {Data: []byte("部品")},
		"other/should_not_appear.md": {Data: []byte("対象外")},
	}

	templates, err := Load(mockFS, "prompts", WithRecursive())
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"top":            "トップ",
		"ja/nested":      "ネスト",
		"ja/_partial":    "部品",
		"en/deep/deeper": "深い",
	}, templates)
}

func TestLoad_WithRecursive_KeepsPrefixFiltering(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/prompt_a.md":    {Data: []byte("A")},
		"prompts/ja/prompt_b.md": {Data: []byte("B")},
		"prompts/ja/other.md":    {Data: []byte("除外")},
	}

	templates, err := Load(mockFS, "prompts", WithPrefix("prompt_"), WithRecursive())
	require.NoError(t, err)

	// 接頭辞はファイル名部分にのみ適用され、ディレクトリ部分は保持される
	assert.Equal(t, map[string]string{
		"a":    "A",
		"ja/b": "B",
	}, templates)
}

func TestLoad_WithExtensions(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/a.md":   {Data: []byte("markdown")},
		"prompts/b.tmpl": {Data: []byte("template")},
		"prompts/c.txt":  {Data: []byte("text")},
		"prompts/d":      {Data: []byte("拡張子なし")},
	}

	t.Run("ドット付き・ドットなしのどちらでも指定できる", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", WithExtensions(".md", "tmpl"))
		require.NoError(t, err)

		assert.Equal(t, map[string]string{"a": "markdown", "b": "template"}, templates)
	})

	t.Run("大文字小文字を区別しない", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", WithExtensions(".MD"))
		require.NoError(t, err)

		assert.Equal(t, map[string]string{"a": "markdown"}, templates)
	})

	t.Run("空文字の指定は無視される", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", WithExtensions("", ".txt"))
		require.NoError(t, err)

		assert.Equal(t, map[string]string{"c": "text"}, templates)
	})

	t.Run("未指定ならすべての拡張子が対象", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts")
		require.NoError(t, err)

		assert.Len(t, templates, 4)
	})
}

// TestLoad_NotExistIsWrapped は、存在しないディレクトリのエラーが
// errors.Is で判定できる形で包まれていることを確認します。
// 呼び出し側が言語ディレクトリの有無を分岐するために依存しています。
func TestLoad_NotExistIsWrapped(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/a.md": {Data: []byte("A")},
	}

	_, err := Load(mockFS, "prompts/en")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Contains(t, err.Error(), "の読み込みに失敗")
}

func TestLoad_NameCollision(t *testing.T) {
	t.Run("拡張子違いで同名になる場合", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"prompts/review.md":   {Data: []byte("md版")},
			"prompts/review.tmpl": {Data: []byte("tmpl版")},
		}

		_, err := Load(mockFS, "prompts")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "テンプレート名が衝突しています")
	})

	t.Run("拡張子で絞り込めば衝突しない", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"prompts/review.md":   {Data: []byte("md版")},
			"prompts/review.tmpl": {Data: []byte("tmpl版")},
		}

		templates, err := Load(mockFS, "prompts", WithExtensions(".md"))
		require.NoError(t, err)
		assert.Equal(t, "md版", templates["review"])
	})

	t.Run("再帰時は別ディレクトリの同名ファイルが衝突しない", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"prompts/ja/review.md": {Data: []byte("日本語")},
			"prompts/en/review.md": {Data: []byte("English")},
		}

		templates, err := Load(mockFS, "prompts", WithRecursive())
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"ja/review": "日本語",
			"en/review": "English",
		}, templates)
	})
}

func TestLoad_RootDirDot(t *testing.T) {
	mockFS := fstest.MapFS{
		"a.md":     {Data: []byte("A")},
		"sub/b.md": {Data: []byte("B")},
	}

	t.Run("カレント指定でも正しく動く", func(t *testing.T) {
		templates, err := Load(mockFS, ".")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"a": "A"}, templates)
	})

	t.Run("カレント指定での再帰", func(t *testing.T) {
		templates, err := Load(mockFS, ".", WithRecursive())
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"a": "A", "sub/b": "B"}, templates)
	})
}

func TestLoad_EmptyResult(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/a.md": {Data: []byte("A")},
	}

	// 該当ファイルが無くてもエラーにはならず、空のマップが返る
	templates, err := Load(mockFS, "prompts", WithPrefix("存在しない_"))
	require.NoError(t, err)
	assert.Empty(t, templates)
}

func TestLoad_RootDirMustBeDirectory(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/a.md": {Data: []byte("A")},
	}

	_, err := Load(mockFS, "prompts/a.md")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotDirectory)
}
