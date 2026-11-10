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
			name:      "front matter and body",
			content:   "---\ndirection: 技術解説\n---\n# 見出し\n\n本文です。\n",
			wantFront: "direction: 技術解説",
			wantBody:  "# 見出し\n\n本文です。\n",
		},
		{
			name:      "no front matter",
			content:   "# 見出し\n\n本文です。\n",
			wantFront: "",
			wantBody:  "# 見出し\n\n本文です。\n",
		},
		{
			name:      "multi-line front matter",
			content:   "---\ndirection: 技術解説\nuse_when: 仕様を説明するとき\n---\n本文\n",
			wantFront: "direction: 技術解説\nuse_when: 仕様を説明するとき",
			wantBody:  "本文\n",
		},
		{
			name:      "closing delimiter at EOF (no body, no newline)",
			content:   "---\ndirection: 技術解説\n---",
			wantFront: "direction: 技術解説",
			wantBody:  "",
		},
		{
			name:      "EOF right after the closing delimiter (no body, newline)",
			content:   "---\ndirection: 技術解説\n---\n",
			wantFront: "direction: 技術解説",
			wantBody:  "",
		},
		{
			name:      "no closing delimiter makes everything body",
			content:   "---\ndirection: 技術解説\n本文\n",
			wantFront: "",
			wantBody:  "---\ndirection: 技術解説\n本文\n",
		},
		{
			name:      "opening delimiter not at line start",
			content:   "本文\n---\ndirection: 技術解説\n---\n",
			wantFront: "",
			wantBody:  "本文\n---\ndirection: 技術解説\n---\n",
		},
		{
			name:      "horizontal rule in body is not a closing delimiter",
			content:   "---\ndirection: 技術解説\n---\n# 見出し\n\n---\n続き\n",
			wantFront: "direction: 技術解説",
			wantBody:  "# 見出し\n\n---\n続き\n",
		},
		{
			name:      "empty front matter block",
			content:   "---\n---\n本文\n",
			wantFront: "",
			wantBody:  "本文\n",
		},
		{
			name:      "empty front matter block with no body",
			content:   "---\n---",
			wantFront: "",
			wantBody:  "",
		},
		{
			name:      "empty string",
			content:   "",
			wantFront: "",
			wantBody:  "",
		},
		{
			name:      "opening delimiter only",
			content:   "---\n",
			wantFront: "",
			wantBody:  "---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body := frontmatter.Split(tt.content)

			assert.Equal(t, tt.wantFront, front, "unexpected front matter")
			assert.Equal(t, tt.wantBody, body, "unexpected body")
		})
	}
}

// TestSplit_ClosingDelimiterIsExactLine confirms that only a line of exactly "---" is a delimiter.
// Accepting a line of another length would cut the front matter off mid-body and break both the
// extracted metadata and the body.
func TestSplit_ClosingDelimiterIsExactLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "four dashes", content: "---\ndirection: 技術解説\n----\n本文\n"},
		{name: "two dashes", content: "---\ndirection: 技術解説\n--\n本文\n"},
		{name: "text after the delimiter", content: "---\ndirection: 技術解説\n--- yaml\n本文\n"},
		{name: "whitespace before the delimiter", content: "---\ndirection: 技術解説\n ---\n本文\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body := frontmatter.Split(tt.content)

			assert.Empty(t, front, "a non-delimiter line was treated as the closing delimiter")
			assert.Equal(t, tt.content, body, "the body was modified")
		})
	}
}

// TestSplit_KeepsBlankLineAfterDelimiter confirms that a blank line right after the closing
// delimiter stays in the body. Only the delimiter line and the newline ending it are removed.
func TestSplit_KeepsBlankLineAfterDelimiter(t *testing.T) {
	front, body := frontmatter.Split("---\ndirection: 技術解説\n---\n\n本文\n")

	assert.Equal(t, "direction: 技術解説", front)
	assert.Equal(t, "\n本文\n", body)
}

// TestSplit_Normalizes confirms that line endings become LF and a leading BOM is removed whether or
// not there is front matter. Both are invisible in an editor, so without normalizing, front matter
// that was written but not detected cannot be tracked down by eye.
func TestSplit_Normalizes(t *testing.T) {
	t.Run("CRLF: with front matter", func(t *testing.T) {
		front, body := frontmatter.Split("---\r\ndirection: 技術解説\r\n---\r\n本文\r\n")

		assert.Equal(t, "direction: 技術解説", front)
		assert.Equal(t, "本文\n", body)
	})

	t.Run("CRLF: without front matter", func(t *testing.T) {
		front, body := frontmatter.Split("# 見出し\r\n本文\r\n")

		assert.Empty(t, front)
		assert.Equal(t, "# 見出し\n本文\n", body)
	})

	t.Run("recognizes front matter after a BOM", func(t *testing.T) {
		front, body := frontmatter.Split("\ufeff---\ndirection: 技術解説\n---\n本文\n")

		assert.Equal(t, "direction: 技術解説", front)
		assert.Equal(t, "本文\n", body)
	})

	t.Run("BOM is not left in the body", func(t *testing.T) {
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
	assert.NotNil(t, bodies, "should return an empty map, not nil")
	assert.NotNil(t, fronts, "should return an empty map, not nil")
}
