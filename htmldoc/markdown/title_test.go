package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ExtractTitle_BlockContext は、行単位の正規表現では
// 誤判定していたケースを検証します。いずれも構文木を辿ることで正しく解決されます。
func TestConverter_ExtractTitle_BlockContext(t *testing.T) {
	c := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "フェンス付きコードブロック内の#は見出しではない",
			input: "```sh\n# これはコメント\necho hi\n```\n\n# 本当のタイトル\n",
			want:  "本当のタイトル",
		},
		{
			name:  "チルダのフェンスでも同様",
			input: "~~~\n# コメント\n~~~\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "4スペースインデントはコードブロックであり見出しではない",
			input: "    # コードブロック内\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "閉じの#列は取り除かれる",
			input: "# タイトル #\n",
			want:  "タイトル",
		},
		{
			name:  "setext形式の見出し（=）",
			input: "タイトル\n=====\n\n本文\n",
			want:  "タイトル",
		},
		{
			name:  "setext形式の見出し（-）",
			input: "サブタイトル\n-----\n",
			want:  "サブタイトル",
		},
		{
			name:  "エスケープされた#は見出しにならない",
			input: "\\# 見出しではない\n\n# 本物\n",
			want:  "本物",
		},
		{
			name:  "強調記法は取り除かれ中身のテキストが残る",
			input: "# **強調**された*タイトル*\n",
			want:  "強調されたタイトル",
		},
		{
			name:  "コードスパンは中身のテキストが残る",
			input: "# `Runner` の説明\n",
			want:  "Runner の説明",
		},
		{
			name:  "リンクはラベルのテキストが残る",
			input: "# [プロジェクト](https://example.com) の概要\n",
			want:  "プロジェクト の概要",
		},
		{
			name:  "引用の中の見出しも見出しとして扱う",
			input: "> # 引用内の見出し\n",
			want:  "引用内の見出し",
		},
		{
			name:  "空入力",
			input: "",
			want:  "",
		},
		{
			name:  "見出しのみで本文がない",
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
	t.Run("WithUnsafeHTML: 無効なら生HTMLは出力されない", func(t *testing.T) {
		c := New(WithUnsafeHTML(false))
		got, err := c.Convert([]byte("<div>raw</div>\n"))
		require.NoError(t, err)
		assert.NotContains(t, string(got), "<div>raw</div>")
	})

	t.Run("WithUnsafeHTML: 有効なら生HTMLがそのまま出力される", func(t *testing.T) {
		c := New(WithUnsafeHTML(true))
		got, err := c.Convert([]byte("<div>raw</div>\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<div>raw</div>")
	})

	t.Run("WithHardWraps: 単純な改行が<br>になる", func(t *testing.T) {
		c := New(WithHardWraps(true))
		got, err := c.Convert([]byte("一行目\n二行目\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<br>")
	})

	t.Run("WithAutoHeadingID: 見出しにid属性が付く", func(t *testing.T) {
		c := New(WithAutoHeadingID(true))
		got, err := c.Convert([]byte("# Section Title\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), `id="section-title"`)
	})

	t.Run("WithAutoHeadingID: 無効ならid属性は付かない", func(t *testing.T) {
		c := New(WithAutoHeadingID(false))
		got, err := c.Convert([]byte("# Section Title\n"))
		require.NoError(t, err)
		assert.NotContains(t, string(got), `id=`)
	})

	t.Run("WithFootnotes: 脚注が展開される", func(t *testing.T) {
		c := New(WithFootnotes(true))
		got, err := c.Convert([]byte("本文[^1]\n\n[^1]: 注釈\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "footnote")
	})

	t.Run("WithTypographer: 引用符が活字記号へ置換される", func(t *testing.T) {
		c := New(WithTypographer(true))
		got, err := c.Convert([]byte("a -- b\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "&ndash;")
	})

	t.Run("複数オプションの併用", func(t *testing.T) {
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

	t.Run("GFMは常に有効", func(t *testing.T) {
		c := New(WithHardWraps(true))
		got, err := c.Convert([]byte("~~打ち消し~~\n"))
		require.NoError(t, err)
		assert.Contains(t, string(got), "<del>")
	})
}

// TestConverter_Concurrent は、1つのConverterを複数goroutineから
// 同時に使用しても安全であることを確認します（go test -race で検証されます）。
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
					t.Errorf("タイトルが一致しません: %q", title)
					return
				}
				got, err := c.Convert(input)
				if err != nil || !strings.Contains(string(got), "<h1>") {
					t.Errorf("変換に失敗しました: %v", err)
					return
				}
			}
		}()
	}

	for range goroutines {
		<-done
	}
}
