package runner_test

import (
	"errors"
	"io"
	"testing"

	"github.com/shouni/go-prompt-kit/mdcast/runner"
)

// --- Mock 定義 (Context を排除した最新の ports 定義に準拠) ---

type mockConverter struct {
	convertFunc      func(markdown []byte) ([]byte, error)
	extractTitleFunc func(markdown []byte) string
}

func (m *mockConverter) Convert(markdown []byte) ([]byte, error) {
	return m.convertFunc(markdown)
}

func (m *mockConverter) ExtractTitleFromMarkdown(markdown []byte) string {
	return m.extractTitleFunc(markdown)
}

type mockRenderer struct {
	renderFunc func(w io.Writer, fragment []byte, locale string, title string) error
}

func (m *mockRenderer) Render(w io.Writer, fragment []byte, locale string, title string) error {
	return m.renderFunc(w, fragment, locale, title)
}

// --- テストケース ---

func TestMarkdownToHtmlRunner_Run(t *testing.T) {
	t.Run("正常系: タイトルが引数で指定されている場合", func(t *testing.T) {
		converter := &mockConverter{
			convertFunc: func(markdown []byte) ([]byte, error) {
				return []byte("<p>hello</p>"), nil
			},
		}
		renderer := &mockRenderer{
			renderFunc: func(w io.Writer, fragment []byte, locale string, title string) error {
				if title != "Specified Title" {
					t.Errorf("期待したタイトルと異なります: %s", title)
				}
				// io.Writer (この場合は bytes.Buffer) への書き込み
				_, _ = w.Write(append([]byte("<html>"), append(fragment, []byte("</html>")...)...))
				return nil
			},
		}

		r := runner.NewMarkdownToHtmlRunner(converter, renderer)
		buf, err := r.Run("Specified Title", []byte("# Header\nContent"))

		if err != nil {
			t.Fatalf("予期せぬエラー: %v", err)
		}
		expected := "<html><p>hello</p></html>"
		if buf.String() != expected {
			t.Errorf("出力内容が不正です。got: %s, want: %s", buf.String(), expected)
		}
	})

	t.Run("正常系: タイトル指定がなく、Markdownから抽出する場合", func(t *testing.T) {
		const extractedTitle = "Extracted Title"
		converter := &mockConverter{
			convertFunc: func(markdown []byte) ([]byte, error) {
				return []byte("fragment"), nil
			},
			extractTitleFunc: func(markdown []byte) string {
				return extractedTitle
			},
		}
		renderer := &mockRenderer{
			renderFunc: func(w io.Writer, fragment []byte, locale string, title string) error {
				if title != extractedTitle {
					t.Errorf("抽出されたタイトルが使用されていません。got: %s, want: %s", title, extractedTitle)
				}
				return nil
			},
		}

		r := runner.NewMarkdownToHtmlRunner(converter, renderer)
		_, err := r.Run("", []byte("# "+extractedTitle))
		if err != nil {
			t.Errorf("エラーが発生しました: %v", err)
		}
	})

	t.Run("境界値: 空のMarkdown入力", func(t *testing.T) {
		// 空入力時は早期リターンするため、モックが呼ばれないことを確認
		r := runner.NewMarkdownToHtmlRunner(&mockConverter{}, &mockRenderer{})
		buf, err := r.Run("Title", []byte(""))

		if err != nil {
			t.Errorf("空入力でエラーが発生しました: %v", err)
		}
		if buf.Len() != 0 {
			t.Errorf("空入力時は空のバッファを期待しましたが、長さ %d でした", buf.Len())
		}
	})

	t.Run("異常系: コンバーターがエラーを返す", func(t *testing.T) {
		expectedErr := errors.New("convert error")
		converter := &mockConverter{
			convertFunc: func(markdown []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}

		r := runner.NewMarkdownToHtmlRunner(converter, &mockRenderer{})
		_, err := r.Run("Title", []byte("data"))

		if !errors.Is(err, expectedErr) {
			t.Errorf("期待したエラー [%v] が返されませんでした: %v", expectedErr, err)
		}
	})

	t.Run("異常系: レンダラーがエラーを返す", func(t *testing.T) {
		converter := &mockConverter{
			convertFunc:      func(markdown []byte) ([]byte, error) { return []byte("ok"), nil },
			extractTitleFunc: func(markdown []byte) string { return "title" },
		}
		expectedErr := errors.New("render error")
		renderer := &mockRenderer{
			renderFunc: func(w io.Writer, fragment []byte, locale string, title string) error {
				return expectedErr
			},
		}

		r := runner.NewMarkdownToHtmlRunner(converter, renderer)
		_, err := r.Run("", []byte("data"))

		if !errors.Is(err, expectedErr) {
			t.Errorf("期待したエラー [%v] が返されませんでした: %v", expectedErr, err)
		}
	})
}
