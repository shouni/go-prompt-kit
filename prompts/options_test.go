package prompts

import (
	"strings"
	"testing"
	"testing/fstest"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithFuncs(t *testing.T) {
	funcs := template.FuncMap{
		"join":  strings.Join,
		"upper": strings.ToUpper,
	}

	t.Run("カスタム関数を呼び出せる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"list": `{{join .Items ", "}}`,
		}, WithFuncs(funcs))
		require.NoError(t, err)

		got, err := b.Build("list", struct{ Items []string }{Items: []string{"a", "b", "c"}})
		require.NoError(t, err)
		assert.Equal(t, "a, b, c", got)
	})

	t.Run("partial からもカスタム関数を呼び出せる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_items": `{{join .Items " / "}}`,
			"main":   `一覧: {{template "_items" .}}`,
		}, WithFuncs(funcs))
		require.NoError(t, err)

		got, err := b.Build("main", struct{ Items []string }{Items: []string{"x", "y"}})
		require.NoError(t, err)
		assert.Equal(t, "一覧: x / y", got)
	})

	t.Run("未登録の関数は解析時にエラーになる", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{"list": `{{join .Items ", "}}`})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `function "join" not defined`)
	})

	t.Run("複数回指定するとマージされる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"m": `{{upper (join .Items "-")}}`,
		},
			WithFuncs(template.FuncMap{"join": strings.Join}),
			WithFuncs(template.FuncMap{"upper": strings.ToUpper}),
		)
		require.NoError(t, err)

		got, err := b.Build("m", struct{ Items []string }{Items: []string{"ab", "cd"}})
		require.NoError(t, err)
		assert.Equal(t, "AB-CD", got)
	})

	t.Run("空のFuncMapは無視される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{"m": "内容"}, WithFuncs(nil))
		require.NoError(t, err)

		got, err := b.Build("m", nil)
		require.NoError(t, err)
		assert.Equal(t, "内容", got)
	})
}

// TestNewBuilder_DuplicateDefinition は、全テンプレートが名前空間を共有することによる
// 静かな上書きを、構築時に検出できることを確認します。
func TestNewBuilder_DuplicateDefinition(t *testing.T) {
	t.Run("複数エントリが同名を define した場合はエラー", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{
			"a": `{{define "shared"}}Aの定義{{end}}A: {{template "shared" .}}`,
			"b": `{{define "shared"}}Bの定義{{end}}B: {{template "shared" .}}`,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateDefinition)
		assert.Contains(t, err.Error(), "'shared'")
		// 衝突した両エントリ名が示されること
		assert.Contains(t, err.Error(), "'a'")
		assert.Contains(t, err.Error(), "'b'")
	})

	t.Run("defineが他エントリのモード名と衝突する場合もエラー", func(t *testing.T) {
		_, err := NewBuilder(map[string]string{
			"a":      `{{define "review"}}乗っ取り{{end}}本文`,
			"review": `本来のレビュー`,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateDefinition)
		assert.Contains(t, err.Error(), "'review'")
	})

	t.Run("1エントリ内でのdefineは正常", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"a": `{{define "shared"}}共通{{end}}A: {{template "shared" .}}`,
			"b": `B: {{template "shared" .}}`,
		})
		require.NoError(t, err)

		gotA, err := b.Build("a", nil)
		require.NoError(t, err)
		assert.Equal(t, "A: 共通", gotA)

		// define されたテンプレートは他エントリからも参照できる
		gotB, err := b.Build("b", nil)
		require.NoError(t, err)
		assert.Equal(t, "B: 共通", gotB)
	})

	t.Run("参照のみでは衝突とみなされない", func(t *testing.T) {
		// 複数のモードが同じ partial を参照するのは正常な使い方
		_, err := NewBuilder(map[string]string{
			"_shared": "共通",
			"a":       `{{template "_shared" .}}`,
			"b":       `{{template "_shared" .}}`,
		})
		require.NoError(t, err)
	})

	t.Run("参照先が後から定義される場合も衝突とみなされない", func(t *testing.T) {
		// "_z" は名前順で "a" より後に処理される
		b, err := NewBuilder(map[string]string{
			"a":  `A: {{template "z_partial" .}}`,
			"_z": `{{define "z_partial"}}後から定義{{end}}`,
		})
		require.NoError(t, err)

		got, err := b.Build("a", nil)
		require.NoError(t, err)
		assert.Equal(t, "A: 後から定義", got)
	})
}

func TestLoadFS_WithFuncs(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/_head.md": {Data: []byte(`{{upper .Title}}`)},
		"prompts/main.md":  {Data: []byte(`{{template "_head" .}} 本文`)},
	}

	b, err := LoadFS(mockFS, "prompts",
		WithFuncs(template.FuncMap{"upper": strings.ToUpper}),
	)
	require.NoError(t, err)

	got, err := b.Build("main", struct{ Title string }{Title: "title"})
	require.NoError(t, err)
	assert.Equal(t, "TITLE 本文", got)
}

func TestLoadFS_CombinedOptions(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/ja/_out.md":  {Data: []byte(`{{join .Tags ","}}`)},
		"prompts/ja/rock.md":  {Data: []byte(`ロック: {{template "ja/_out" .}}`)},
		"prompts/ja/note.txt": {Data: []byte(`除外される`)},
	}

	b, err := LoadFS(mockFS, "prompts",
		WithRecursive(),
		WithExtensions(".md"),
		WithFuncs(template.FuncMap{"join": strings.Join}),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"ja/rock"}, b.Modes())

	got, err := b.Build("ja/rock", struct{ Tags []string }{Tags: []string{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, "ロック: a,b", got)
}

func TestWithDefaultMode(t *testing.T) {
	templates := map[string]string{
		"default": "既定の内容",
		"rock":    "ロックの内容",
		"_hidden": "部品",
	}

	t.Run("未知のモードは既定モードへ委ねられる", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode("default"))
		require.NoError(t, err)

		got, err := b.Build("存在しないモード", nil)
		require.NoError(t, err)
		assert.Equal(t, "既定の内容", got)
	})

	t.Run("空文字も既定モードへ委ねられる", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode("default"))
		require.NoError(t, err)

		got, err := b.Build("", nil)
		require.NoError(t, err)
		assert.Equal(t, "既定の内容", got)
	})

	t.Run("登録済みのモードはそのまま使われる", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode("default"))
		require.NoError(t, err)

		got, err := b.Build("rock", nil)
		require.NoError(t, err)
		assert.Equal(t, "ロックの内容", got)
	})

	t.Run("partial名を渡した場合も既定モードへ委ねられる", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode("default"))
		require.NoError(t, err)

		got, err := b.Build("_hidden", nil)
		require.NoError(t, err)
		assert.Equal(t, "既定の内容", got)
	})

	t.Run("未指定なら従来どおりErrUnknownMode", func(t *testing.T) {
		b, err := NewBuilder(templates)
		require.NoError(t, err)

		_, err = b.Build("", nil)
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("実在しない既定モードは構築時に検出される", func(t *testing.T) {
		_, err := NewBuilder(templates, WithDefaultMode("typo"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownMode)
		assert.Contains(t, err.Error(), "既定モード 'typo' が登録されていません")
	})

	t.Run("partialを既定モードには指定できない", func(t *testing.T) {
		_, err := NewBuilder(templates, WithDefaultMode("_hidden"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("空文字の指定は無視される", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode(""))
		require.NoError(t, err)

		_, err = b.Build("unknown", nil)
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("Has は既定モードへのフォールバックを反映しない", func(t *testing.T) {
		b, err := NewBuilder(templates, WithDefaultMode("default"))
		require.NoError(t, err)

		assert.True(t, b.Has("rock"))
		assert.False(t, b.Has("存在しないモード"), "Has は実際の登録状況を返すべきです")
	})
}

func TestWithPartialPrefix(t *testing.T) {
	t.Run("接頭辞を変更できる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"@shared": "共通ルール",
			"review":  `本文: {{template "@shared" .}}`,
		}, WithPartialPrefix("@"))
		require.NoError(t, err)

		assert.Equal(t, []string{"review"}, b.Modes())

		got, err := b.Build("review", nil)
		require.NoError(t, err)
		assert.Equal(t, "本文: 共通ルール", got)
	})

	t.Run("変更後は既定の _ がモードとして公開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_legacy": "旧来はpartial扱いだった内容",
			"review":  "本文",
		}, WithPartialPrefix("@"))
		require.NoError(t, err)

		assert.Equal(t, []string{"_legacy", "review"}, b.Modes())

		got, err := b.Build("_legacy", nil)
		require.NoError(t, err)
		assert.Equal(t, "旧来はpartial扱いだった内容", got)
	})

	t.Run("空文字を指定すると全エントリがモードになる", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_a": "A",
			"b":  "B",
		}, WithPartialPrefix(""))
		require.NoError(t, err)

		assert.Equal(t, []string{"_a", "b"}, b.Modes())
	})

	t.Run("既定は DefaultPartialPrefix", func(t *testing.T) {
		assert.Equal(t, "_", DefaultPartialPrefix)

		b, err := NewBuilder(map[string]string{
			"_a": "A",
			"b":  "B",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"b"}, b.Modes())
	})

	t.Run("パス形式でも末尾要素で判定される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"ja/@out": "出力",
			"ja/rock": `{{template "ja/@out" .}}`,
		}, WithPartialPrefix("@"))
		require.NoError(t, err)

		assert.Equal(t, []string{"ja/rock"}, b.Modes())
	})
}

// TestLoadFS_ComicStylePattern は、ap-comic 相当の構成
// （工程ごとのディレクトリ・.md のみ・既定モードへのフォールバック）を再現します。
func TestLoadFS_ComicStylePattern(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/outline/default.md": {Data: []byte("章立て: {{.Title}}")},
		"prompts/outline/notes.txt":  {Data: []byte("除外される")},
	}

	b, err := LoadFS(mockFS, "prompts/outline",
		WithExtensions(".md"),
		WithDefaultMode("default"),
	)
	require.NoError(t, err)

	// モード未指定でも既定へ落ちる
	got, err := b.Build("", struct{ Title string }{Title: "第一章"})
	require.NoError(t, err)
	assert.Equal(t, "章立て: 第一章", got)
}
