package builder_test

import (
	"errors"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/md/builder"
	"github.com/shouni/go-prompt-kit/md/jsonconverter"
	"github.com/shouni/go-prompt-kit/md/renderer"
)

// TestBuilder_WithConverter_JSONPipeline は、Markdown以外のConverterを注入して
// パイプラインを組めることを確認します（従来は手動配線が必要でした）。
func TestBuilder_WithConverter_JSONPipeline(t *testing.T) {
	tpl := template.Must(template.New("fragment").Parse(
		`<h1>{{.title}}</h1><p>{{.body}}</p>`))

	b, err := builder.New(
		builder.WithConverter(jsonconverter.New(tpl)),
		builder.WithRendererOptions(renderer.WithCSS("body{margin:0}")),
		builder.WithLang("en"),
	)
	require.NoError(t, err)

	r, err := b.BuildRunner()
	require.NoError(t, err)

	buf, err := r.Run("", []byte(`{"title":"レビュー結果","body":"本文"}`))
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, `<html lang="en">`)
	// タイトルはJSONの "title" キーから解決される
	assert.Contains(t, got, "<title>レビュー結果</title>")
	assert.Contains(t, got, "<h1>レビュー結果</h1>")
	assert.Contains(t, got, "<p>本文</p>")
	assert.Contains(t, got, "body{margin:0}")
	// 既定のCSSへ差し替わっていること
	assert.NotContains(t, got, "--color-primary-main")
}

func TestBuilder_WithRenderer(t *testing.T) {
	stub := &stubRenderer{}

	b, err := builder.New(builder.WithRenderer(stub))
	require.NoError(t, err)

	r, err := b.BuildRunner()
	require.NoError(t, err)

	_, err = r.Run("タイトル", []byte("# 見出し\n"))
	require.NoError(t, err)

	assert.True(t, stub.called, "注入したRendererが使用されていません")
	assert.Equal(t, "タイトル", stub.title)
}

func TestBuilder_ConverterOptions(t *testing.T) {
	b, err := builder.New(builder.WithEnableUnsafeHTML(true))
	require.NoError(t, err)

	r, err := b.BuildRunner()
	require.NoError(t, err)

	buf, err := r.Run("T", []byte("<div>raw</div>\n"))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "<div>raw</div>")
}

func TestBuilder_DefaultTitle(t *testing.T) {
	b, err := builder.New(builder.WithDefaultTitle("無題のドキュメント"))
	require.NoError(t, err)

	r, err := b.BuildRunner()
	require.NoError(t, err)

	// 見出しのない入力なので既定タイトルへフォールバックする
	buf, err := r.Run("", []byte("本文のみです。\n"))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "<title>無題のドキュメント</title>")
}

func TestBuilder_UnsupportedMode(t *testing.T) {
	b, err := builder.New(builder.WithMode("pdf"))
	require.NoError(t, err)

	_, err = b.BuildRunner()
	require.Error(t, err)
	assert.ErrorIs(t, err, builder.ErrUnsupportedMode)
	assert.Contains(t, err.Error(), "pdf")
}

func TestBuilder_RendererOptionError(t *testing.T) {
	_, err := builder.New(builder.WithRendererOptions(renderer.WithTemplateText(`{{.Unclosed`)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rendererの初期化エラー")
}

// TestBuilder_EndToEnd_Markdown は、既定構成でMarkdownから完全なHTMLが生成されることを確認します。
func TestBuilder_EndToEnd_Markdown(t *testing.T) {
	b, err := builder.New()
	require.NoError(t, err)

	r, err := b.BuildRunner()
	require.NoError(t, err)

	buf, err := r.Run("", []byte("```\n# コメント\n```\n\n# 実際のタイトル\n\n本文です。\n"))
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, `<html lang="ja-jp">`)
	// コードブロック内の見出しではなく、実際の見出しがタイトルになる
	assert.Contains(t, got, "<title>実際のタイトル</title>")
	assert.Contains(t, got, "<h1>実際のタイトル</h1>")
	assert.True(t, strings.Contains(got, "<pre>"), "コードブロックが出力されていません")
}

// stubRenderer は、呼び出しの有無と引数を記録する ports.Renderer です。
type stubRenderer struct {
	called bool
	title  string
}

func (s *stubRenderer) Render(w io.Writer, bodyHTML []byte, _, title string) error {
	s.called = true
	s.title = title
	if _, err := w.Write(bodyHTML); err != nil {
		return errors.New("書き込みに失敗")
	}
	return nil
}
