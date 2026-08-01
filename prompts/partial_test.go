package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder_Partials(t *testing.T) {
	t.Run("partialがモード本文から展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_output": "出力形式: JSON",
			"review":  "レビューしてください。\n{{template \"_output\" .}}",
		})
		require.NoError(t, err)

		got, err := b.Build("review", nil)
		require.NoError(t, err)
		assert.Equal(t, "レビューしてください。\n出力形式: JSON", got)
	})

	t.Run("partialへデータを渡せる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_greeting": "こんにちは、{{.Name}}さん",
			"main":      "{{template \"_greeting\" .}}！",
		})
		require.NoError(t, err)

		got, err := b.Build("main", struct{ Name string }{Name: "太郎"})
		require.NoError(t, err)
		assert.Equal(t, "こんにちは、太郎さん！", got)
	})

	t.Run("partialへ部分的なデータを渡せる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_item": "- {{.Label}}",
			"list":  "一覧:\n{{template \"_item\" .First}}\n{{template \"_item\" .Second}}",
		})
		require.NoError(t, err)

		type item struct{ Label string }
		data := struct {
			First  item
			Second item
		}{
			First:  item{Label: "あ"},
			Second: item{Label: "い"},
		}

		got, err := b.Build("list", data)
		require.NoError(t, err)
		assert.Equal(t, "一覧:\n- あ\n- い", got)
	})

	t.Run("partialは複数のモードから共有される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_shared": "共通ルール",
			"a":       "A: {{template \"_shared\" .}}",
			"bb":      "B: {{template \"_shared\" .}}",
		})
		require.NoError(t, err)

		gotA, err := b.Build("a", nil)
		require.NoError(t, err)
		assert.Equal(t, "A: 共通ルール", gotA)

		gotB, err := b.Build("bb", nil)
		require.NoError(t, err)
		assert.Equal(t, "B: 共通ルール", gotB)
	})

	t.Run("partialはモードとして公開されない", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_output": "出力形式",
			"review":  "本文",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"review"}, b.Modes())
		assert.False(t, b.Has("_output"))

		_, err = b.Build("_output", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("パス形式のキーでも末尾要素でpartialを判定する", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"en/_output": "Output: JSON",
			"en/review":  "Review this.\n{{template \"en/_output\" .}}",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"en/review"}, b.Modes())

		got, err := b.Build("en/review", nil)
		require.NoError(t, err)
		assert.Equal(t, "Review this.\nOutput: JSON", got)
	})

	t.Run("存在しないpartialを参照した場合は解析時にエラーになる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"review": "{{template \"_missing\" .}}",
		})
		require.NoError(t, err, "未定義テンプレートの参照は実行時に検出される")

		_, err = b.Build("review", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "プロンプトテンプレートの実行に失敗しました")
	})

	t.Run("partialしかない場合はエラーになる", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{"_only": "内容"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyTemplates)
	})
}

func TestBuilder_Modes(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"zebra":   "z",
		"alpha":   "a",
		"middle":  "m",
		"_hidden": "h",
	})
	require.NoError(t, err)

	t.Run("名前順で返る", func(t *testing.T) {
		assert.Equal(t, []string{"alpha", "middle", "zebra"}, b.Modes())
	})

	t.Run("返り値を書き換えても内部状態は壊れない", func(t *testing.T) {
		modes := b.Modes()
		modes[0] = "書き換え"
		assert.Equal(t, []string{"alpha", "middle", "zebra"}, b.Modes())
	})
}

func TestBuilder_Has(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"review":  "r",
		"_hidden": "h",
	})
	require.NoError(t, err)

	assert.True(t, b.Has("review"))
	assert.False(t, b.Has("_hidden"))
	assert.False(t, b.Has("unknown"))
}

func TestNewBuilder_SentinelErrors(t *testing.T) {
	t.Run("空マップはErrEmptyTemplates", func(t *testing.T) {
		_, err := NewBuilder(nil)
		assert.ErrorIs(t, err, ErrEmptyTemplates)

		_, err = NewBuilder(map[string]string{})
		assert.ErrorIs(t, err, ErrEmptyTemplates)
	})

	t.Run("未知のモードはErrUnknownMode", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{"mode": "内容"})
		require.NoError(t, err)

		_, err = b.Build("unknown", nil)
		assert.ErrorIs(t, err, ErrUnknownMode)
		// 既存の呼び出し元が文字列で判定している場合に備え、メッセージも維持する
		assert.Contains(t, err.Error(), "不明なモードです: 'unknown'")
	})

	t.Run("空の内容はエラー", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{"mode": ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "内容が空です")
	})

	t.Run("解析できないテンプレートはエラー", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{"mode": "{{.Unclosed"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "プロンプト 'mode' の解析に失敗")
	})
}
