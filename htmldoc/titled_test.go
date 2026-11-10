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

// countingConverter is a Converter that counts how many times each method is called.
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

// titledCounter is countingConverter plus ports.TitledConverter.
type titledCounter struct{ countingConverter }

func (c *titledCounter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	c.titled++
	return []byte("<p>" + string(input) + "</p>"), "抽出タイトル", nil
}

// TestDocument_Run_UsesTitledConverter confirms that when the title is extracted automatically,
// TitledConverter is called once and the input is not parsed twice.
func TestDocument_Run_UsesTitledConverter(t *testing.T) {
	c := &titledCounter{}
	doc, err := htmldoc.New(htmldoc.WithConverter(c))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, doc.Run(&buf, "", []byte("本文")))

	assert.Equal(t, 1, c.titled, "the single-parse path was not used")
	assert.Zero(t, c.converts)
	assert.Zero(t, c.extracts, "the input was parsed twice")
	assert.Contains(t, buf.String(), "<title>抽出タイトル</title>")
}

// TestDocument_Run_TitleGivenSkipsExtraction confirms that no extraction happens at all when a
// title is given.
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

// TestDocument_Run_FallsBackToConverter confirms that a Converter implementing only ports.Converter
// still works through the Convert and ExtractTitle path.
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

// emptyTitleConverter is a TitledConverter that cannot extract a title.
type emptyTitleConverter struct{ countingConverter }

func (c *emptyTitleConverter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	c.titled++
	return []byte("<p>" + string(input) + "</p>"), "", nil
}

// TestDocument_Run_TitledConverterFallsBackToDefault confirms that the single-parse path also falls
// back to the default when no title can be extracted.
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

// plainConverter hides ConvertWithTitle and exposes only Converter.
type plainConverter struct{ inner ports.Converter }

func (p plainConverter) Convert(input []byte) ([]byte, error) { return p.inner.Convert(input) }
func (p plainConverter) ExtractTitle(input []byte) string     { return p.inner.ExtractTitle(input) }

// TestDocument_Run_BothPathsAgree confirms that the single-parse path and the other path output the
// same document. Speed must not change the result.
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
