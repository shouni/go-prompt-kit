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

// TestNew_WithConverter_JSONPipeline confirms that a pipeline can be built by injecting a Converter
// other than Markdown.
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
	// The title resolves from the JSON "title" key
	assert.Contains(t, got, "<title>レビュー結果</title>")
	assert.Contains(t, got, "<h1>レビュー結果</h1>")
	assert.Contains(t, got, "<p>本文</p>")
	assert.Contains(t, got, "body{margin:0}")
	// The default CSS is in place
	assert.NotContains(t, got, "--color-primary-main")
}

func TestNew_WithRenderer(t *testing.T) {
	stub := &stubRenderer{}

	doc, err := htmldoc.New(htmldoc.WithRenderer(stub))
	require.NoError(t, err)

	require.NoError(t, doc.Run(io.Discard, "タイトル", []byte("# 見出し\n")))

	assert.True(t, stub.called, "the injected Renderer was not used")
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

	// The input has no heading, so it falls back to the default title
	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文のみです。\n")))
	assert.Contains(t, buf.String(), "<title>無題のドキュメント</title>")
}

func TestNew_RendererOptionError(t *testing.T) {
	_, err := htmldoc.New(htmldoc.WithRendererOptions(renderer.WithTemplateText(`{{.Unclosed`)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initialize renderer")
}

// TestNew_EndToEnd_Markdown confirms that the default setup generates complete HTML from Markdown.
func TestNew_EndToEnd_Markdown(t *testing.T) {
	doc, err := htmldoc.New()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("```\n# コメント\n```\n\n# 実際のタイトル\n\n本文です。\n")))

	got := buf.String()
	assert.Contains(t, got, `<html lang="ja-jp">`)
	// The actual heading, not the one inside the code block, becomes the title
	assert.Contains(t, got, "<title>実際のタイトル</title>")
	assert.Contains(t, got, "<h1>実際のタイトル</h1>")
	assert.True(t, strings.Contains(got, "<pre>"), "the code block was not output")
}

// stubRenderer is a ports.Renderer that records whether it was called and with what.
type stubRenderer struct {
	called bool
	title  string
}

func (s *stubRenderer) Render(w io.Writer, bodyHTML []byte, _, title string) error {
	s.called = true
	s.title = title
	if _, err := w.Write(bodyHTML); err != nil {
		return errors.New("write failed")
	}
	return nil
}
