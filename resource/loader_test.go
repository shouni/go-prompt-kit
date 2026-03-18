package resource

import (
	"testing"
	"testing/fstest"
)

func TestLoad(t *testing.T) {
	// メモリ上の仮想ファイルシステムを作成
	mockFS := fstest.MapFS{
		"prompts/prompt_review.md":   {Data: []byte("Review Template")},
		"prompts/prompt_release.md":  {Data: []byte("Release Template")},
		"prompts/other_file.txt":     {Data: []byte("Ignore Me")},
		"other_dir/prompt_unused.md": {Data: []byte("Wrong Dir")},
	}

	t.Run("正常系: プロンプトが正しくロードされること", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", "prompt_")
		if err != nil {
			t.Fatalf("Load 失敗: %v", err)
		}

		if len(templates) != 2 {
			t.Errorf("期待されるテンプレート数は 2 ですが、%d でした", len(templates))
		}

		if templates["review"] != "Review Template" {
			t.Errorf("review の内容が不正です: %s", templates["review"])
		}
	})

	t.Run("異常系: 存在しないディレクトリを指定した場合エラーになること", func(t *testing.T) {
		_, err := Load(mockFS, "non_existent", "prompt_")
		if err == nil {
			t.Error("存在しないディレクトリでエラーが発生しませんでした")
		}
	})
}
