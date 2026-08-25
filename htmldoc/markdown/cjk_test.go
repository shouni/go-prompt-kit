package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_WithCJK は、東アジア言語向けの改行処理を確認します。
// 段落中のソフト改行は既定でHTMLに残り、ブラウザ上で空白として描画されるため、
// 日本語では文の途中に空白が入ります。
func TestConverter_WithCJK(t *testing.T) {
	const japanese = "これは日本語の文章で、\n途中で改行しています。\n"

	tests := []struct {
		name   string
		enable bool
		input  string
		want   string
	}{
		{
			name:  "無効なら改行が残る",
			input: japanese,
			want:  "<p>これは日本語の文章で、\n途中で改行しています。</p>\n",
		},
		{
			name:   "有効なら全角文字間の改行が取り除かれる",
			enable: true,
			input:  japanese,
			want:   "<p>これは日本語の文章で、途中で改行しています。</p>\n",
		},
		{
			name:   "有効でも英文の改行は空白として残る",
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

// TestConverter_WithCJK_TitleUnaffected は、CJK拡張がタイトル抽出へ影響しないことを
// 確認します。CJK拡張が変えるのはレンダラーの改行の扱いだけで、タイトルは構文木から
// 収集するため、複数行にまたがる見出しの語の区切りは拡張の有無にかかわらず空白のまま残ります。
func TestConverter_WithCJK_TitleUnaffected(t *testing.T) {
	c := New(WithCJK(true))

	fragment, title, err := c.ConvertWithTitle([]byte("# 日本語のタイトル\n\n本文です、\n続きます。\n"))
	require.NoError(t, err)

	assert.Equal(t, "日本語のタイトル", title)
	assert.Contains(t, string(fragment), "<h1>日本語のタイトル</h1>")
	assert.Contains(t, string(fragment), "<p>本文です、続きます。</p>")
}

// TestConverter_ExtractTitle_MultilineHeading は、複数行の見出し（setext形式）で
// 語が繋がらないよう空白が補われることを、CJK拡張の有無にかかわらず固定します。
func TestConverter_ExtractTitle_MultilineHeading(t *testing.T) {
	const src = "日本語の\nタイトル\n=========\n"

	assert.Equal(t, "日本語の タイトル", New().ExtractTitle([]byte(src)))
	assert.Equal(t, "日本語の タイトル", New(WithCJK(true)).ExtractTitle([]byte(src)))
}
