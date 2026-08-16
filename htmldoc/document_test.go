package htmldoc_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/htmldoc"
)

// --- Mock 定義 ---

type mockConverter struct {
	convertFunc      func(input []byte) ([]byte, error)
	extractTitleFunc func(input []byte) string
}

func (m *mockConverter) Convert(input []byte) ([]byte, error) {
	return m.convertFunc(input)
}

func (m *mockConverter) ExtractTitle(input []byte) string {
	return m.extractTitleFunc(input)
}

type mockRenderer struct {
	renderFunc func(w io.Writer, fragment []byte, lang string, title string) error
}

func (m *mockRenderer) Render(w io.Writer, fragment []byte, lang string, title string) error {
	return m.renderFunc(w, fragment, lang, title)
}

// newDocument は、モックを注入した Document を構築します。
func newDocument(t *testing.T, c *mockConverter, r *mockRenderer, opts ...htmldoc.Option) *htmldoc.Document {
	t.Helper()

	opts = append([]htmldoc.Option{
		htmldoc.WithConverter(c),
		htmldoc.WithRenderer(r),
	}, opts...)

	doc, err := htmldoc.New(opts...)
	require.NoError(t, err)

	return doc
}

// --- テストケース ---

func TestDocument_Run(t *testing.T) {
	t.Run("正常系: タイトルが引数で指定されている場合", func(t *testing.T) {
		converter := &mockConverter{
			convertFunc: func(_ []byte) ([]byte, error) {
				return []byte("<p>hello</p>"), nil
			},
		}
		renderer := &mockRenderer{
			renderFunc: func(w io.Writer, fragment []byte, _ string, title string) error {
				assert.Equal(t, "Specified Title", title)
				_, err := w.Write(append([]byte("<html>"), append(fragment, []byte("</html>")...)...))
				return err
			},
		}

		var buf bytes.Buffer
		doc := newDocument(t, converter, renderer)
		require.NoError(t, doc.Run(&buf, "Specified Title", []byte("# Header\nContent")))

		assert.Equal(t, "<html><p>hello</p></html>", buf.String())
	})

	t.Run("正常系: タイトル指定がなく、入力から抽出する場合", func(t *testing.T) {
		const extractedTitle = "Extracted Title"
		converter := &mockConverter{
			convertFunc:      func(_ []byte) ([]byte, error) { return []byte("fragment"), nil },
			extractTitleFunc: func(_ []byte) string { return extractedTitle },
		}
		renderer := &mockRenderer{
			renderFunc: func(_ io.Writer, _ []byte, _ string, title string) error {
				assert.Equal(t, extractedTitle, title, "抽出されたタイトルが使用されていません")
				return nil
			},
		}

		doc := newDocument(t, converter, renderer)
		require.NoError(t, doc.Run(io.Discard, "", []byte("# "+extractedTitle)))
	})

	t.Run("境界値: 空の入力", func(t *testing.T) {
		// 空入力時は早期リターンするため、モックが呼ばれないことを確認します。
		var buf bytes.Buffer
		doc := newDocument(t, &mockConverter{}, &mockRenderer{})

		require.NoError(t, doc.Run(&buf, "Title", nil))
		assert.Zero(t, buf.Len(), "空入力時は何も書き出さないはずです")
	})

	t.Run("異常系: コンバーターがエラーを返す", func(t *testing.T) {
		expectedErr := errors.New("convert error")
		converter := &mockConverter{
			convertFunc: func(_ []byte) ([]byte, error) { return nil, expectedErr },
		}

		doc := newDocument(t, converter, &mockRenderer{})
		err := doc.Run(io.Discard, "Title", []byte("data"))

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系: レンダラーがエラーを返す", func(t *testing.T) {
		expectedErr := errors.New("render error")
		converter := &mockConverter{
			convertFunc:      func(_ []byte) ([]byte, error) { return []byte("ok"), nil },
			extractTitleFunc: func(_ []byte) string { return "title" },
		}
		renderer := &mockRenderer{
			renderFunc: func(_ io.Writer, _ []byte, _ string, _ string) error { return expectedErr },
		}

		doc := newDocument(t, converter, renderer)
		err := doc.Run(io.Discard, "", []byte("data"))

		assert.ErrorIs(t, err, expectedErr)
	})
}
