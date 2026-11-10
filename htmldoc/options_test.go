package htmldoc_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/htmldoc"
)

// captured records the arguments passed to the Renderer.
type captured struct {
	lang  string
	title string
}

// newCapturingDocument builds a Document that records the arguments passed to the Renderer.
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

	assert.Equal(t, "ja-jp", got.lang, "the default language was not used")
	assert.Equal(t, "Document", got.title, "the default title was not used")
}

func TestDocument_WithLang(t *testing.T) {
	t.Run("uses the given language", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithLang("en-US"))

		require.NoError(t, doc.Run(io.Discard, "T", []byte("input")))
		assert.Equal(t, "en-US", got.lang)
	})

	t.Run("empty string keeps the default", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithLang(""))

		require.NoError(t, doc.Run(io.Discard, "T", []byte("input")))
		assert.Equal(t, "ja-jp", got.lang)
	})
}

func TestDocument_WithDefaultTitle(t *testing.T) {
	t.Run("uses the default title when none is extracted", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "無題", got.title)
	})

	t.Run("extracted title wins", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "抽出タイトル", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "抽出タイトル", got.title)
	})

	t.Run("argument wins over everything", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "抽出タイトル", htmldoc.WithDefaultTitle("無題"))

		require.NoError(t, doc.Run(io.Discard, "引数タイトル", []byte("input")))
		assert.Equal(t, "引数タイトル", got.title)
	})

	t.Run("empty string keeps the default", func(t *testing.T) {
		doc, got := newCapturingDocument(t, "", htmldoc.WithDefaultTitle(""))

		require.NoError(t, doc.Run(io.Discard, "", []byte("input")))
		assert.Equal(t, "Document", got.title)
	})
}
