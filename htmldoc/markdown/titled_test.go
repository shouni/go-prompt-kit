package markdown

import (
	"testing"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ConvertWithTitle confirms that the single-parse path gives the same result as
// calling Convert and ExtractTitle separately. ports.TitledConverter promises this, so a drift
// would change the output merely depending on whether a title was given.
func TestConverter_ConvertWithTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "with heading", input: "# タイトル\n\n本文です。\n"},
		{name: "without heading", input: "本文だけの文書です。\n"},
		{name: "setext heading", input: "タイトル\n=====\n\n本文です。\n"},
		{name: "heading marker in a code block", input: "```\n# これは見出しではない\n```\n\n## 本当の見出し\n"},
		{name: "heading with inline markup", input: "# **強調**と[link](https://example.com)\n"},
		{name: "empty input", input: ""},
	}

	c := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.input)

			wantFragment, err := c.Convert(input)
			require.NoError(t, err)
			wantTitle := c.ExtractTitle(input)

			gotFragment, gotTitle, err := c.ConvertWithTitle(input)
			require.NoError(t, err)

			assert.Equal(t, string(wantFragment), string(gotFragment))
			assert.Equal(t, wantTitle, gotTitle)
		})
	}
}

// TestConverter_ImplementsTitledConverter confirms that the optimized path is actually in a shape
// that gets chosen. Document keeps working even when the interface is no longer satisfied, so the
// drift would only be noticed after it had silently gone back to parsing twice.
func TestConverter_ImplementsTitledConverter(t *testing.T) {
	var titled ports.TitledConverter = New()

	fragment, title, err := titled.ConvertWithTitle([]byte("# 見出し\n"))
	require.NoError(t, err)
	assert.Equal(t, "見出し", title)
	assert.Contains(t, string(fragment), "<h1>見出し</h1>")
}
