package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ExtractTitle_BlockContext verifies cases that a line-based regular expression used
// to get wrong. Walking the syntax tree resolves all of them correctly.
func TestConverter_ExtractTitle_BlockContext(t *testing.T) {
	c := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "# in a fenced code block is not a heading",
			input: "```sh\n# これはコメント\necho hi\n```\n\n# 本当のタイトル\n",
			want:  "本当のタイトル",
		},
		{
			name:  "same with a tilde fence",
			input: "~~~\n# コメント\n~~~\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "4-space indent is a code block, not a heading",
			input: "    # コードブロック内\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "closing hashes are removed",
			input: "# タイトル #\n",
			want:  "タイトル",
		},
		{
			name:  "setext heading (=)",
			input: "タイトル\n=====\n\n本文\n",
			want:  "タイトル",
		},
		{
			name:  "setext heading (-)",
			input: "サブタイトル\n-----\n",
			want:  "サブタイトル",
		},
		{
			name:  "escaped # is not a heading",
			input: "\\# 見出しではない\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "emphasis is stripped to its text",
			input: "# **強調**された*タイトル*\n",
			want:  "強調されたタイトル",
		},
		{
			name:  "code span keeps its text",
			input: "# `Runner` の説明\n",
			want:  "Runner の説明",
		},
		{
			name:  "link keeps its label text",
			input: "# [プロジェクト](https://example.com) の概要\n",
			want:  "プロジェクト の概要",
		},
		{
			name:  "heading inside a blockquote counts",
			input: "> # 引用内の見出し\n",
			want:  "引用内の見出し",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "heading only, no body",
			input: "# 単独の見出し",
			want:  "単独の見出し",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.ExtractTitle([]byte(tt.input)))
		})
	}
}

func TestConverter_Options(t *testing.T) {
	t.Run("WithUnsafeHTML: disabled omits raw HTML", func(t *testing.T) {
		c := New(WithUnsafeHTML(false))
		got, err := c.Convert([]byte("<div>raw</div>\n"))
		require.NoError(t, err)
		assert.NotContains(t, string(got), "<div>raw</div>")
	})

	t.Run("WithUnsafeHTML: enabled outputs raw HTML as is", func(t *testing.T) {
		c := New(WithUnsafeHTML(true))
		got, err := c.Convert([]byte("<div>raw</div>\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<div>raw</div>")
	})

	t.Run("WithHardWraps: plain newline becomes <br>", func(t *testing.T) {
		c := New(WithHardWraps(true))
		got, err := c.Convert([]byte("一行目\n二行目\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<br>")
	})

	t.Run("WithAutoHeadingID: headings get an id", func(t *testing.T) {
		c := New(WithAutoHeadingID(true))
		got, err := c.Convert([]byte("# Section Title\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), `id="section-title"`)
	})

	t.Run("WithAutoHeadingID: disabled adds no id", func(t *testing.T) {
		c := New(WithAutoHeadingID(false))
		got, err := c.Convert([]byte("# Section Title\n"))
		require.NoError(t, err)
		assert.NotContains(t, string(got), `id=`)
	})

	t.Run("WithFootnotes: footnotes are rendered", func(t *testing.T) {
		c := New(WithFootnotes(true))
		got, err := c.Convert([]byte("本文[^1]\n\n[^1]: 注釈\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "footnote")
	})

	t.Run("WithTypographer: quotes become typographic", func(t *testing.T) {
		c := New(WithTypographer(true))
		got, err := c.Convert([]byte("a -- b\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "&ndash;")
	})

	t.Run("several options combined", func(t *testing.T) {
		c := New(
			WithUnsafeHTML(true),
			WithHardWraps(true),
			WithAutoHeadingID(true),
		)
		got, err := c.Convert([]byte("# Title\n<span>x</span>\nnext\n"))
		require.NoError(t, err)

		out := string(got)
		assert.Contains(t, out, `id="title"`)
		assert.Contains(t, out, "<span>x</span>")
		assert.Contains(t, out, "<br>")
	})

	t.Run("GFM is always on", func(t *testing.T) {
		c := New(WithHardWraps(true))
		got, err := c.Convert([]byte("~~打ち消し~~\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<del>")
	})
}

// TestConverter_Concurrent confirms that one Converter is safe to use from several goroutines at
// once (verified by go test -race).
func TestConverter_Concurrent(t *testing.T) {
	c := New()
	input := []byte("# 並行タイトル\n\n本文です。\n")

	const goroutines = 8
	done := make(chan struct{}, goroutines)

	for range goroutines {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 20 {
				if title := c.ExtractTitle(input); title != "並行タイトル" {
					t.Errorf("title mismatch: %q", title)
					return
				}
				got, err := c.Convert(input)
				if err != nil || !strings.Contains(string(got), "<h1>") {
					t.Errorf("conversion failed: %v", err)
					return
				}
			}
		}()
	}

	for range goroutines {
		<-done
	}
}
