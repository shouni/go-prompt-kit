package prompts

import (
	"testing"
	"testing/fstest"

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
		// どのモードが無かったのかを文面に残す（番兵だけでは特定できないため）
		assert.Contains(t, err.Error(), "'unknown'")
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

func TestNewBuilder_TrimPartials(t *testing.T) {
	// 末尾が改行の partial を、本文の途中と末尾の両方から参照します。
	// 途中で参照した側にだけ差が出るのが、このオプションが扱う現象です。
	const partial = "共通の一文。\n"
	templates := map[string]string{
		"_shared": partial,
		"middle":  "前の段落。\n{{template \"_shared\" .}}\n後の段落。",
		"tail":    "本文。\n{{template \"_shared\" .}}",
	}

	tests := []struct {
		name       string
		opts       []Option
		wantMiddle string
		wantTail   string
	}{
		{
			name:       "既定では partial の末尾の改行がそのまま入る",
			wantMiddle: "前の段落。\n共通の一文。\n\n後の段落。",
			wantTail:   "本文。\n共通の一文。\n",
		},
		{
			name:       "WithTrimPartials は末尾の改行を取り除く",
			opts:       []Option{WithTrimPartials()},
			wantMiddle: "前の段落。\n共通の一文。\n後の段落。",
			wantTail:   "本文。\n共通の一文。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBuilder(templates, tt.opts...)
			require.NoError(t, err)

			middle, err := b.Build("middle", nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMiddle, middle)

			tail, err := b.Build("tail", nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTail, tail)
		})
	}
}

func TestNewBuilder_TrimPartialsScope(t *testing.T) {
	t.Run("モード本文の末尾の改行は残る", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_shared": "部品。\n",
			"main":    "本文。\n{{template \"_shared\" .}}末尾。\n",
		}, WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Build("main", nil)
		require.NoError(t, err)
		assert.Equal(t, "本文。\n部品。末尾。\n", got)
	})

	t.Run("行末の空白と本文中の改行は変えない", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_shared": "1行目  \n2行目  \n\n",
			"main":    "{{template \"_shared\" .}}",
		}, WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Build("main", nil)
		require.NoError(t, err)
		assert.Equal(t, "1行目  \n2行目  ", got)
	})

	t.Run("CRLF の末尾も取り除く", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_shared": "部品。\r\n",
			"main":    "{{template \"_shared\" .}}続き。",
		}, WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Build("main", nil)
		require.NoError(t, err)
		assert.Equal(t, "部品。続き。", got)
	})

	t.Run("改行だけの partial は空の内容として扱わない", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_blank": "\n",
			"main":   "前{{template \"_blank\" .}}後",
		}, WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Build("main", nil)
		require.NoError(t, err)
		assert.Equal(t, "前後", got)
	})

	t.Run("Expand も取り除いた本文で展開する", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_shared": "部品。\n",
			"main":    "{{template \"_shared\" .}}続き{{.Field}}。",
		}, WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Expand("main")
		require.NoError(t, err)
		assert.Equal(t, "部品。続き{{.Field}}。", got)
	})

	t.Run("LoadFS からも指定できる", func(t *testing.T) {
		files := fstest.MapFS{
			"prompts/_shared.md": &fstest.MapFile{Data: []byte("部品。\n")},
			"prompts/main.md":    &fstest.MapFile{Data: []byte("{{template \"_shared\" .}}続き。\n")},
		}

		b, err := LoadFS(files, "prompts", WithTrimPartials())
		require.NoError(t, err)

		got, err := b.Build("main", nil)
		require.NoError(t, err)
		assert.Equal(t, "部品。続き。\n", got)
	})
}
