package htmldoc_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shouni/go-prompt-kit/htmldoc"
	"github.com/shouni/go-prompt-kit/htmldoc/markdown"
	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingConverter は、どのメソッドが何回呼ばれたかを数える Converter です。
type countingConverter struct {
	converts int
	extracts int
	titled   int
}

func (c *countingConverter) Convert(input []byte) ([]byte, error) {
	c.converts++
	return []byte("<p>" + string(input) + "</p>"), nil
}

func (c *countingConverter) ExtractTitle([]byte) string {
	c.extracts++
	return "抽出タイトル"
}

// titledCounter は countingConverter に ports.TitledConverter を足したものです。
type titledCounter struct{ countingConverter }

func (c *titledCounter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	c.titled++
	return []byte("<p>" + string(input) + "</p>"), "抽出タイトル", nil
}

// TestDocument_Run_UsesTitledConverter は、タイトルを自動抽出するとき
// TitledConverter が1回だけ呼ばれ、入力が二度解析されないことを確認します。
func TestDocument_Run_UsesTitledConverter(t *testing.T) {
	c := &titledCounter{}
	doc, err := htmldoc.New(htmldoc.WithConverter(c))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文")))

	assert.Equal(t, 1, c.titled, "まとめて解析する経路が使われていません")
	assert.Zero(t, c.converts)
	assert.Zero(t, c.extracts, "入力を二度解析しています")
	assert.Contains(t, buf.String(), "<title>抽出タイトル</title>")
}

// TestDocument_Run_TitleGivenSkipsExtraction は、タイトルが指定されている場合に
// 抽出そのものを行わないことを確認します。
func TestDocument_Run_TitleGivenSkipsExtraction(t *testing.T) {
	c := &titledCounter{}
	doc, err := htmldoc.New(htmldoc.WithConverter(c))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "指定タイトル", []byte("本文")))

	assert.Equal(t, 1, c.converts)
	assert.Zero(t, c.titled)
	assert.Zero(t, c.extracts)
	assert.Contains(t, buf.String(), "<title>指定タイトル</title>")
}

// TestDocument_Run_FallsBackToConverter は、ports.Converter だけを実装した
// Converter でもこれまでどおり動くことを確認します。
func TestDocument_Run_FallsBackToConverter(t *testing.T) {
	c := &countingConverter{}
	doc, err := htmldoc.New(htmldoc.WithConverter(c))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文")))

	assert.Equal(t, 1, c.converts)
	assert.Equal(t, 1, c.extracts)
	assert.Contains(t, buf.String(), "<title>抽出タイトル</title>")
}

// emptyTitleConverter は、タイトルを抽出できない TitledConverter です。
type emptyTitleConverter struct{ countingConverter }

func (c *emptyTitleConverter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	c.titled++
	return []byte("<p>" + string(input) + "</p>"), "", nil
}

// TestDocument_Run_TitledConverterFallsBackToDefault は、まとめて解析する経路でも
// タイトルを抽出できなければ既定値になることを確認します。
func TestDocument_Run_TitledConverterFallsBackToDefault(t *testing.T) {
	doc, err := htmldoc.New(
		htmldoc.WithConverter(&emptyTitleConverter{}),
		htmldoc.WithDefaultTitle("既定タイトル"),
	)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文")))

	assert.Contains(t, buf.String(), "<title>既定タイトル</title>")
}

// plainConverter は ConvertWithTitle を隠し、Converter だけを見せます。
type plainConverter struct{ inner ports.Converter }

func (p plainConverter) Convert(input []byte) ([]byte, error) { return p.inner.Convert(input) }
func (p plainConverter) ExtractTitle(input []byte) string     { return p.inner.ExtractTitle(input) }

// TestDocument_Run_BothPathsAgree は、まとめて解析する経路とそうでない経路が
// 同じドキュメントを出力することを確認します。速さのために結果が変わっては困ります。
func TestDocument_Run_BothPathsAgree(t *testing.T) {
	inputs := []string{
		"# タイトル\n\n本文です。\n",
		"見出しのない本文です。\n",
		"タイトル\n====\n\n- 項目\n- 項目\n",
	}

	fast, err := htmldoc.New()
	require.NoError(t, err)
	slow, err := htmldoc.New(htmldoc.WithConverter(plainConverter{markdown.New()}))
	require.NoError(t, err)

	for _, input := range inputs {
		t.Run(strings.SplitN(input, "\n", 2)[0], func(t *testing.T) {
			var fastBuf, slowBuf bytes.Buffer
			require.NoError(t, fast.Run(&fastBuf, "", []byte(input)))
			require.NoError(t, slow.Run(&slowBuf, "", []byte(input)))

			assert.Equal(t, slowBuf.String(), fastBuf.String())
		})
	}
}
