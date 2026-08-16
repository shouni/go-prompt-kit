package htmldoc_test

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/htmldoc"
	"github.com/shouni/go-prompt-kit/htmldoc/jsondoc"
	"github.com/shouni/go-prompt-kit/htmldoc/markdown"
	"github.com/shouni/go-prompt-kit/htmldoc/renderer"
)

// TestNew_WithConverter_JSONPipeline は、Markdown以外のConverterを注入して
// パイプラインを組めることを確認します。
func TestNew_WithConverter_JSONPipeline(t *testing.T) {
	tpl := template.Must(template.New("fragment").Parse(
		`<h1>{{.title}}</h1><p>{{.body}}</p>`))

	doc, err := htmldoc.New(
		htmldoc.WithConverter(jsondoc.New(tpl)),
		htmldoc.WithRendererOptions(renderer.WithCSS("body{margin:0}")),
		htmldoc.WithLang("en"),
	)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte(`{"title":"レビュー結果","body":"本文"}`)))

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

func TestNew_WithRenderer(t *testing.T) {
	stub := &stubRenderer{}

	doc, err := htmldoc.New(htmldoc.WithRenderer(stub))
	require.NoError(t, err)

	require.NoError(t, doc.Run(io.Discard, "タイトル", []byte("# 見出し\n")))

	assert.True(t, stub.called, "注入したRendererが使用されていません")
	assert.Equal(t, "タイトル", stub.title)
}

func TestNew_ConverterOptions(t *testing.T) {
	doc, err := htmldoc.New(
		htmldoc.WithConverterOptions(markdown.WithUnsafeHTML(true)),
	)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "T", []byte("<div>raw</div>\n")))
	assert.Contains(t, buf.String(), "<div>raw</div>")
}

func TestNew_DefaultTitle(t *testing.T) {
	doc, err := htmldoc.New(htmldoc.WithDefaultTitle("無題のドキュメント"))
	require.NoError(t, err)

	// 見出しのない入力なので既定タイトルへフォールバックする
	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文のみです。\n")))
	assert.Contains(t, buf.String(), "<title>無題のドキュメント</title>")
}

func TestNew_RendererOptionError(t *testing.T) {
	_, err := htmldoc.New(htmldoc.WithRendererOptions(renderer.WithTemplateText(`{{.Unclosed`)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rendererの初期化エラー")
}

// TestNew_EndToEnd_Markdown は、既定構成でMarkdownから完全なHTMLが生成されることを確認します。
func TestNew_EndToEnd_Markdown(t *testing.T) {
	doc, err := htmldoc.New()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("```\n# コメント\n```\n\n# 実際のタイトル\n\n本文です。\n")))

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
