package jsondoc

import (
	"testing"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ConvertWithTitle confirms that the single-decode path gives the same result as
// calling Convert and ExtractTitle separately.
func TestConverter_ConvertWithTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "with title", input: `{"title":"タイトル","body":"本文"}`},
		{name: "without title", input: `{"body":"本文"}`},
		{name: "non-string title", input: `{"title":123,"body":"本文"}`},
		{name: "top-level array", input: `[1,2,3]`},
		{name: "empty input", input: ""},
	}

	c := New(mustParse(t, "test", `<p>{{if .}}ok{{end}}</p>`))
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

// TestConverter_ExtractTitle_Streaming pins the boundaries of the implementation that stops reading
// as soon as the key is found.
func TestConverter_ExtractTitle_Streaming(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		input string
		want  string
	}{
		{name: "first key", input: `{"title":"T","body":"b"}`, want: "T"},
		{name: "key in the middle", input: `{"a":1,"b":[1,2],"title":"T"}`, want: "T"},
		{name: "nested key ignored", input: `{"a":{"title":"inner"}}`, want: ""},
		{name: "key missing", input: `{"a":1}`, want: ""},
		{name: "non-string value", input: `{"title":123}`, want: ""},
		{name: "empty object", input: `{}`, want: ""},
		{name: "top-level array", input: `[{"title":"T"}]`, want: ""},
		{name: "empty input", input: "", want: ""},
		{name: "invalid JSON", input: `{invalid`, want: ""},
		{name: "extracts when broken after the key", input: `{"title":"T", invalid`, want: "T"},
		{name: "does not extract when broken before the key", input: `{invalid, "title":"T"}`, want: ""},
		{name: "custom key", key: "name", input: `{"title":"T","name":"N"}`, want: "N"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.key != "" {
				opts = append(opts, WithTitleKey(tt.key))
			}
			c := New(mustParse(t, "test", `{{.}}`), opts...)

			assert.Equal(t, tt.want, c.ExtractTitle([]byte(tt.input)))
		})
	}
}

// TestConverter_ImplementsTitledConverter confirms that the optimized path is actually in a shape
// that gets chosen.
func TestConverter_ImplementsTitledConverter(t *testing.T) {
	var titled ports.TitledConverter = New(mustParse(t, "test", `<h1>{{.title}}</h1>`))

	fragment, title, err := titled.ConvertWithTitle([]byte(`{"title":"タイトル"}`))
	require.NoError(t, err)
	assert.Equal(t, "タイトル", title)
	assert.Equal(t, "<h1>タイトル</h1>", string(fragment))
}
