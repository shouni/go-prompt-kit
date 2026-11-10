package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_WithCJK checks the line-break handling for East Asian languages. By default a soft
// line break inside a paragraph stays in the HTML and a browser renders it as a space, which puts a
// space in the middle of a Japanese sentence.
func TestConverter_WithCJK(t *testing.T) {
	const japanese = "これは日本語の文章で、\n途中で改行しています。\n"

	tests := []struct {
		name   string
		enable bool
		input  string
		want   string
	}{
		{
			name:  "disabled keeps newlines",
			input: japanese,
			want:  "<p>これは日本語の文章で、\n途中で改行しています。</p>\n",
		},
		{
			name:   "enabled removes newlines between full-width characters",
			enable: true,
			input:  japanese,
			want:   "<p>これは日本語の文章で、途中で改行しています。</p>\n",
		},
		{
			name:   "enabled keeps English newlines as spaces",
			enable: true,
			input:  "This is English\ntext across lines.\n",
			want:   "<p>This is English\ntext across lines.</p>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(WithCJK(tt.enable)).Convert([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

// TestConverter_WithCJK_TitleUnaffected confirms that the CJK extension does not affect title
// extraction. The extension only changes how the renderer treats line breaks, while the title is
// collected from the syntax tree, so the word separator in a heading spanning several lines stays a
// space with or without the extension.
func TestConverter_WithCJK_TitleUnaffected(t *testing.T) {
	c := New(WithCJK(true))

	fragment, title, err := c.ConvertWithTitle([]byte("# 日本語のタイトル\n\n本文です、\n続きます。\n"))
	require.NoError(t, err)

	assert.Equal(t, "日本語のタイトル", title)
	assert.Contains(t, string(fragment), "<h1>日本語のタイトル</h1>")
	assert.Contains(t, string(fragment), "<p>本文です、続きます。</p>")
}

// TestConverter_ExtractTitle_MultilineHeading pins, with or without the CJK extension, that a space
// is added in a multi-line (setext) heading so that words do not run together.
func TestConverter_ExtractTitle_MultilineHeading(t *testing.T) {
	const src = "日本語の\nタイトル\n=========\n"

	assert.Equal(t, "日本語の タイトル", New().ExtractTitle([]byte(src)))
	assert.Equal(t, "日本語の タイトル", New(WithCJK(true)).ExtractTitle([]byte(src)))
}
