package frontmatter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shouni/go-prompt-kit/frontmatter"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFront string
		wantBody  string
	}{
		{
			name:      "front matter と本文",
			content:   "---\ndirection: 技術解説\n---\n# 見出し\n\n本文です。\n",
			wantFront: "direction: 技術解説",
			wantBody:  "# 見出し\n\n本文です。\n",
		},
		{
			name:      "front matter なし",
			content:   "# 見出し\n\n本文です。\n",
			wantFront: "",
			wantBody:  "# 見出し\n\n本文です。\n",
		},
		{
			name:      "複数行の front matter",
			content:   "---\ndirection: 技術解説\nuse_when: 仕様を説明するとき\n---\n本文\n",
			wantFront: "direction: 技術解説\nuse_when: 仕様を説明するとき",
			wantBody:  "本文\n",
		},
		{
			name:      "終了の区切りがファイル末尾（本文なし・改行なし）",
			content:   "---\ndirection: 技術解説\n---",
			wantFront: "direction: 技術解説",
			wantBody:  "",
		},
		{
			name:      "終了の区切りの直後がファイル末尾（本文なし・改行あり）",
			content:   "---\ndirection: 技術解説\n---\n",
			wantFront: "direction: 技術解説",
			wantBody:  "",
		},
		{
			name:      "終了の区切りが無い場合は全体を本文として扱う",
			content:   "---\ndirection: 技術解説\n本文\n",
			wantFront: "",
			wantBody:  "---\ndirection: 技術解説\n本文\n",
		},
		{
			name:      "開始の区切りが行頭にない",
			content:   "本文\n---\ndirection: 技術解説\n---\n",
			wantFront: "",
			wantBody:  "本文\n---\ndirection: 技術解説\n---\n",
		},
		{
			name:      "本文中の水平線は終了の区切りとみなさない",
			content:   "---\ndirection: 技術解説\n---\n# 見出し\n\n---\n続き\n",
			wantFront: "direction: 技術解説",
			wantBody:  "# 見出し\n\n---\n続き\n",
		},
		{
			name:      "中身のない front matter ブロック",
			content:   "---\n---\n本文\n",
			wantFront: "",
			wantBody:  "本文\n",
		},
		{
			name:      "中身も本文もない front matter ブロック",
			content:   "---\n---",
			wantFront: "",
			wantBody:  "",
		},
		{
			name:      "空文字",
			content:   "",
			wantFront: "",
			wantBody:  "",
		},
		{
			name:      "開始の区切りだけ",
			content:   "---\n",
			wantFront: "",
			wantBody:  "---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body := frontmatter.Split(tt.content)

			assert.Equal(t, tt.wantFront, front, "front matter が期待と異なります")
			assert.Equal(t, tt.wantBody, body, "本文が期待と異なります")
		})
	}
}

// TestSplit_ClosingDelimiterIsExactLine は、区切りとみなす行を "---" だけに限る
// ことを確認します。文字数の違う行を区切りとして受け入れると、front matter が
// 本文の途中で打ち切られ、切り出したメタデータも本文も壊れます。
func TestSplit_ClosingDelimiterIsExactLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "ダッシュ4本", content: "---\ndirection: 技術解説\n----\n本文\n"},
		{name: "ダッシュ2本", content: "---\ndirection: 技術解説\n--\n本文\n"},
		{name: "区切りの後ろに文字がある", content: "---\ndirection: 技術解説\n--- yaml\n本文\n"},
		{name: "区切りの前に空白がある", content: "---\ndirection: 技術解説\n ---\n本文\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body := frontmatter.Split(tt.content)

			assert.Empty(t, front, "区切りではない行を終了とみなしています")
			assert.Equal(t, tt.content, body, "本文が加工されています")
		})
	}
}

// TestSplit_KeepsBlankLineAfterDelimiter は、終了の区切りの直後にある空行を
// 本文へ残すことを確認します。取り除くのは区切り行とその行末の改行だけです。
func TestSplit_KeepsBlankLineAfterDelimiter(t *testing.T) {
	front, body := frontmatter.Split("---\ndirection: 技術解説\n---\n\n本文\n")

	assert.Equal(t, "direction: 技術解説", front)
	assert.Equal(t, "\n本文\n", body)
}

// TestSplit_Normalizes は、front matter の有無にかかわらず改行が LF へ揃い、
// 先頭のBOMが取り除かれることを確認します。どちらもエディタ上では見えないため、
// 揃えずにいると front matter を書いたのに認識されない状態を目視で追えません。
func TestSplit_Normalizes(t *testing.T) {
	t.Run("CRLF: front matter あり", func(t *testing.T) {
		front, body := frontmatter.Split("---\r\ndirection: 技術解説\r\n---\r\n本文\r\n")

		assert.Equal(t, "direction: 技術解説", front)
		assert.Equal(t, "本文\n", body)
	})

	t.Run("CRLF: front matter なし", func(t *testing.T) {
		front, body := frontmatter.Split("# 見出し\r\n本文\r\n")

		assert.Empty(t, front)
		assert.Equal(t, "# 見出し\n本文\n", body)
	})

	t.Run("BOM付きでも front matter を認識する", func(t *testing.T) {
		front, body := frontmatter.Split("\ufeff---\ndirection: 技術解説\n---\n本文\n")

		assert.Equal(t, "direction: 技術解説", front)
		assert.Equal(t, "本文\n", body)
	})

	t.Run("BOMは本文にも残さない", func(t *testing.T) {
		front, body := frontmatter.Split("\ufeff# 見出し\n")

		assert.Empty(t, front)
		assert.Equal(t, "# 見出し\n", body)
	})
}

func TestSplitMap(t *testing.T) {
	files := map[string]string{
		"tech_solo": "---\ndirection: 技術解説\n---\n本文A\n",
		"news":      "本文B\n",
		"_writing":  "---\n---\n本文C\n",
	}

	bodies, fronts := frontmatter.SplitMap(files)

	assert.Equal(t, map[string]string{
		"tech_solo": "本文A\n",
		"news":      "本文B\n",
		"_writing":  "本文C\n",
	}, bodies)
	assert.Equal(t, map[string]string{
		"tech_solo": "direction: 技術解説",
		"news":      "",
		"_writing":  "",
	}, fronts)
}

func TestSplitMap_Empty(t *testing.T) {
	bodies, fronts := frontmatter.SplitMap(nil)

	assert.Empty(t, bodies)
	assert.Empty(t, fronts)
	assert.NotNil(t, bodies, "nil ではなく空のマップを返すべきです")
	assert.NotNil(t, fronts, "nil ではなく空のマップを返すべきです")
}
