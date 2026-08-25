package markdown

import (
	"testing"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ConvertWithTitle は、まとめて解析する経路の結果が
// Convert と ExtractTitle をそれぞれ呼んだ場合と一致することを確認します。
// ports.TitledConverter がこの一致を約束しているため、ずれると
// タイトルの指定有無だけで出力が変わります。
func TestConverter_ConvertWithTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "見出しあり", input: "# タイトル\n\n本文です。\n"},
		{name: "見出しなし", input: "本文だけの文書です。\n"},
		{name: "setext見出し", input: "タイトル\n=====\n\n本文です。\n"},
		{name: "コードブロック内の見出し記号", input: "```\n# これは見出しではない\n```\n\n## 本当の見出し\n"},
		{name: "装飾を含む見出し", input: "# **強調**と[link](https://example.com)\n"},
		{name: "空入力", input: ""},
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

// TestConverter_ImplementsTitledConverter は、最適化された経路が
// 実際に選ばれる形になっていることを確認します。
// インターフェースを満たさなくなっても Document は動いてしまい、
// 気付けるのは黙って2回解析に戻ったあとだからです。
func TestConverter_ImplementsTitledConverter(t *testing.T) {
	var titled ports.TitledConverter = New()

	fragment, title, err := titled.ConvertWithTitle([]byte("# 見出し\n"))
	require.NoError(t, err)
	assert.Equal(t, "見出し", title)
	assert.Contains(t, string(fragment), "<h1>見出し</h1>")
}
