package runner_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/md/runner"
)

// captured は、Renderer へ渡された引数を記録します。
type captured struct {
	lang  string
	title string
}

func newCapturingRunner(t *testing.T, extracted string, opts ...runner.Option) (*runner.DocumentRunner, *captured) {
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

	return runner.NewDocumentRunner(converter, renderer, opts...), got
}

func TestDocumentRunner_Defaults(t *testing.T) {
	r, got := newCapturingRunner(t, "")

	_, err := r.Run("", []byte("input"))
	require.NoError(t, err)

	assert.Equal(t, "ja-jp", got.lang, "既定の言語が使用されていません")
	assert.Equal(t, "Document", got.title, "既定のタイトルが使用されていません")
}

func TestDocumentRunner_WithLang(t *testing.T) {
	t.Run("指定した言語が使用される", func(t *testing.T) {
		r, got := newCapturingRunner(t, "", runner.WithLang("en-US"))

		_, err := r.Run("T", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "en-US", got.lang)
	})

	t.Run("空文字を渡した場合は既定値が維持される", func(t *testing.T) {
		r, got := newCapturingRunner(t, "", runner.WithLang(""))

		_, err := r.Run("T", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "ja-jp", got.lang)
	})
}

func TestDocumentRunner_WithDefaultTitle(t *testing.T) {
	t.Run("抽出できない場合に指定した既定タイトルが使われる", func(t *testing.T) {
		r, got := newCapturingRunner(t, "", runner.WithDefaultTitle("無題"))

		_, err := r.Run("", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "無題", got.title)
	})

	t.Run("抽出できる場合は抽出結果が優先される", func(t *testing.T) {
		r, got := newCapturingRunner(t, "抽出タイトル", runner.WithDefaultTitle("無題"))

		_, err := r.Run("", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "抽出タイトル", got.title)
	})

	t.Run("引数指定が最優先される", func(t *testing.T) {
		r, got := newCapturingRunner(t, "抽出タイトル", runner.WithDefaultTitle("無題"))

		_, err := r.Run("引数タイトル", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "引数タイトル", got.title)
	})

	t.Run("空文字を渡した場合は既定値が維持される", func(t *testing.T) {
		r, got := newCapturingRunner(t, "", runner.WithDefaultTitle(""))

		_, err := r.Run("", []byte("input"))
		require.NoError(t, err)
		assert.Equal(t, "Document", got.title)
	})
}

// acceptsDocumentRunner は、旧型名 MarkdownToHTMLRunner が
// DocumentRunner の別名であることをコンパイル時に確認するためのヘルパーです。
func acceptsDocumentRunner(r *runner.MarkdownToHTMLRunner) *runner.DocumentRunner {
	return r
}

// TestMarkdownToHTMLRunner_Alias は、旧名の型エイリアスと
// 非推奨コンストラクタが引き続き利用できることを確認します。
func TestMarkdownToHTMLRunner_Alias(t *testing.T) {
	converter := &mockConverter{
		convertFunc:      func(_ []byte) ([]byte, error) { return []byte("<p>x</p>"), nil },
		extractTitleFunc: func(_ []byte) string { return "" },
	}
	renderer := &mockRenderer{
		renderFunc: func(w io.Writer, fragment []byte, _ string, _ string) error {
			_, err := w.Write(fragment)
			return err
		},
	}

	// 旧コンストラクタの戻り値がそのまま新しい型として扱えること
	r := acceptsDocumentRunner(runner.NewMarkdownToHTMLRunner(converter, renderer))
	require.NotNil(t, r)

	buf, err := r.Run("T", []byte("input"))
	require.NoError(t, err)
	assert.Equal(t, "<p>x</p>", buf.String())
}
