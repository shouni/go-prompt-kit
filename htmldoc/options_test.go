package htmldoc_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/htmldoc"
)

// captured は、Renderer へ渡された引数を記録します。
type captured struct {
	lang  string
	title string
}

// newCapturingDocument は、Rendererへの引数を記録する Document を構築します。
func newCapturingDocument(t *testing.T, extracted string, opts ...htmldoc.Option) (*htmldoc.Document, *captured) {
	t.Helper()

	got := &captured{}
	converter := &mockConverter{
		convertFunc:      func(_ []byte) ([]byte, error) { return []byte("<p>x</p>"), nil },
		extractTitleFunc: func(_ []byte) string { return extracted },
	}
	renderer := &mockRenderer{
		renderFunc: func(_ io.Writer, _ []byte, lang string, title string) error {
			got.lang = lang
			got.title = title
			return nil
		},
	}

	return newDocument(t, converter, renderer, opts...), got
}

func TestDocument_Defaults(t *testing.T) {
	doc, got := newCapturingDocument(t, "")

	require.NoError(t, doc.Run(io.Discard, "", []byte("input")))

	assert.Equal(t, "ja-jp", got.lang, "既定の言語が使用されていません")
	assert.Equal(t, "Document", got.title, "既定のタイトルが使用されていません")
}

func TestDocument_WithLang(t *testing.T) {
	t.Run("指定した言語が使用される", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithLang("en-US"))

		require.NoError(t, doc.Run(io.Discard, "T", []byte("input")))
		assert.Equal(t, "en-US", got.lang)
	})

	t.Run("空文字を渡した場合は既定値が維持される", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithLang(""))

		require.NoError(t, doc.Run(io.Discard, "T", []byte("input")))
		assert.Equal(t, "ja-jp", got.lang)
	})
}

func TestDocument_WithDefaultTitle(t *testing.T) {
	t.Run("抽出できない場合に指定した既定タイトルが使われる", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "無題", got.title)
	})

	t.Run("抽出できる場合は抽出結果が優先される", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "抽出タイトル", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "抽出タイトル", got.title)
	})

	t.Run("引数指定が最優先される", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "抽出タイトル", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "引数タイトル", []byte("input")))
		assert.Equal(t, "引数タイトル", got.title)
	})

	t.Run("空文字を渡した場合は既定値が維持される", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithDefaultTitle(""))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "Document", got.title)
	})
}
