package renderer

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer_Defaults(t *testing.T) {
	r, err := New()
	require.NoError(t, err)
	require.NotNil(t, r)

	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, []byte("<p>本文</p>"), "ja", "既定タイトル"))

	got := buf.String()
	assert.Contains(t, got, "<!DOCTYPE html>")
	assert.Contains(t, got, `<html lang="ja">`)
	assert.Contains(t, got, "<title>既定タイトル</title>")
	assert.Contains(t, got, "<p>本文</p>")
	// 埋め込みの default.css が差し込まれていること
	assert.Contains(t, got, "--color-primary-main")
}

func TestNewRenderer_WithCSS(t *testing.T) {
	t.Run("任意のCSSへ差し替えられる", func(t *testing.T) {
		r, err := New(WithCSS("body { color: rebeccapurple; }"))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>x</p>"), "ja", "T"))

		got := buf.String()
		assert.Contains(t, got, "body { color: rebeccapurple; }")
		// 既定のCSSは含まれない
		assert.NotContains(t, got, "--color-primary-main")
	})

	t.Run("空文字を渡すとスタイルなしになる", func(t *testing.T) {
		r, err := New(WithCSS(""))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>x</p>"), "ja", "T"))

		assert.Contains(t, buf.String(), "<style></style>")
	})
}

func TestNewRenderer_WithTemplateText(t *testing.T) {
	t.Run("任意のテンプレートへ差し替えられる", func(t *testing.T) {
		r, err := New(
			WithTemplateText(`<article lang="{{.Lang}}"><h1>{{.Title}}</h1>{{.Content}}</article>`),
			WithCSS("ignored"),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>本文</p>"), "en", "Title"))

		got := buf.String()
		assert.Equal(t, `<article lang="en"><h1>Title</h1><p>本文</p></article>`, got)
		// テンプレートが .Style を参照していないためCSSは出力されない
		assert.NotContains(t, got, "ignored")
	})

	t.Run("パースに失敗した場合はエラーを返す", func(t *testing.T) {
		_, err := New(WithTemplateText(`{{.Unclosed`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTMLテンプレートのパースエラー")
	})
}

func TestNewRenderer_WithTemplate(t *testing.T) {
	t.Run("パース済みテンプレートを渡せる", func(t *testing.T) {
		tpl := template.Must(template.New("custom").Parse(`{{.Title}}|{{.Content}}`))

		r, err := New(WithTemplate(tpl))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<b>b</b>"), "ja", "T"))
		assert.Equal(t, "T|<b>b</b>", buf.String())
	})

	t.Run("nilを渡した場合はエラーを返す", func(t *testing.T) {
		_, err := New(WithTemplate(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "テンプレートがnilです")
	})
}

func TestRenderer_Render_Error(t *testing.T) {
	t.Run("テンプレート実行に失敗した場合はエラーを返す", func(t *testing.T) {
		// TemplateData に存在しないフィールドを参照させる
		r, err := New(WithTemplateText(`{{.DoesNotExist}}`))
		require.NoError(t, err)

		var buf bytes.Buffer
		err = r.Render(&buf, []byte("x"), "ja", "T")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTMLテンプレートの実行エラー")
	})

	t.Run("書き込み先がエラーを返す場合", func(t *testing.T) {
		r, err := New()
		require.NoError(t, err)

		err = r.Render(failingWriter{}, []byte("x"), "ja", "T")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTMLテンプレートの実行エラー")
	})
}

// failingWriter は常に書き込みエラーを返す io.Writer です。
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, assert.AnError
}

func TestRenderer_Render_EscapesTitle(t *testing.T) {
	r, err := New(WithCSS(""))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, []byte("<p>ok</p>"), "ja", `<script>alert(1)</script>`))

	got := buf.String()
	// タイトルはエスケープされる（string型のため）
	assert.NotContains(t, got, "<title><script>")
	assert.True(t, strings.Contains(got, "&lt;script&gt;") || strings.Contains(got, "\\u003cscript\\u003e"),
		"タイトルがエスケープされていません: %s", got)
	// 本文フラグメントは template.HTML のためエスケープされない
	assert.Contains(t, got, "<p>ok</p>")
}

func TestNewRenderer_WithExtraCSS(t *testing.T) {
	t.Run("既定CSSの後ろへ連結される", func(t *testing.T) {
		r, err := New(WithExtraCSS(".finding { color: red; }"))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>x</p>"), "ja", "T"))

		got := buf.String()
		// 既定CSSが残っていること
		assert.Contains(t, got, "--color-primary-main")
		// 追加分も含まれること
		assert.Contains(t, got, ".finding { color: red; }")
		// 追加分が後ろにあること（同じセレクタなら上書きできる）
		assert.Less(t, strings.Index(got, "--color-primary-main"), strings.Index(got, ".finding"))
	})

	t.Run("style ブロックは1つだけになる", func(t *testing.T) {
		r, err := New(WithExtraCSS("body { color: red; }"))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>x</p>"), "ja", "T"))

		assert.Equal(t, 1, strings.Count(buf.String(), "<style>"))
	})

	t.Run("WithCSSと併用すると指定した土台の後ろへ足される", func(t *testing.T) {
		r, err := New(
			WithCSS("body { margin: 0; }"),
			WithExtraCSS(".x { color: blue; }"),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte("<p>x</p>"), "ja", "T"))

		got := buf.String()
		assert.Contains(t, got, "body { margin: 0; }")
		assert.Contains(t, got, ".x { color: blue; }")
		assert.NotContains(t, got, "--color-primary-main")
		assert.Less(t, strings.Index(got, "body { margin: 0; }"), strings.Index(got, ".x {"))
	})

	t.Run("複数回指定すると指定順に連結される", func(t *testing.T) {
		r, err := New(
			WithCSS("/*base*/"),
			WithExtraCSS(".a {}"),
			WithExtraCSS(".b {}"),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte(""), "ja", "T"))

		got := buf.String()
		assert.Less(t, strings.Index(got, "/*base*/"), strings.Index(got, ".a {}"))
		assert.Less(t, strings.Index(got, ".a {}"), strings.Index(got, ".b {}"))
	})

	t.Run("空文字は無視される", func(t *testing.T) {
		r, err := New(WithCSS("body{}"), WithExtraCSS(""))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte(""), "ja", "T"))
		assert.Contains(t, buf.String(), "<style>body{}</style>")
	})

	t.Run("土台が空なら追加分だけになる", func(t *testing.T) {
		r, err := New(WithCSS(""), WithExtraCSS(".only {}"))
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, r.Render(&buf, []byte(""), "ja", "T"))
		assert.Contains(t, buf.String(), "<style>.only {}</style>")
	})
}
