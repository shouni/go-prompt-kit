package jsondoc

import (
	"testing"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_ConvertWithTitle は、まとめてデコードする経路の結果が
// Convert と ExtractTitle をそれぞれ呼んだ場合と一致することを確認します。
func TestConverter_ConvertWithTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "titleあり", input: `{"title":"タイトル","body":"本文"}`},
		{name: "titleなし", input: `{"body":"本文"}`},
		{name: "titleが文字列でない", input: `{"title":123,"body":"本文"}`},
		{name: "トップレベルが配列", input: `[1,2,3]`},
		{name: "空入力", input: ""},
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

// TestConverter_ExtractTitle_Streaming は、キーが見つかった時点で読み取りを
// 打ち切る実装の境界を固定します。
func TestConverter_ExtractTitle_Streaming(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		input string
		want  string
	}{
		{name: "先頭のキー", input: `{"title":"T","body":"b"}`, want: "T"},
		{name: "途中のキー", input: `{"a":1,"b":[1,2],"title":"T"}`, want: "T"},
		{name: "入れ子は対象外", input: `{"a":{"title":"inner"}}`, want: ""},
		{name: "キーが無い", input: `{"a":1}`, want: ""},
		{name: "値が文字列でない", input: `{"title":123}`, want: ""},
		{name: "空のオブジェクト", input: `{}`, want: ""},
		{name: "トップレベルが配列", input: `[{"title":"T"}]`, want: ""},
		{name: "空入力", input: "", want: ""},
		{name: "不正なJSON", input: `{invalid`, want: ""},
		{name: "キーの後ろが壊れていても取り出せる", input: `{"title":"T", invalid`, want: "T"},
		{name: "キーの手前が壊れていれば取り出せない", input: `{invalid, "title":"T"}`, want: ""},
		{name: "キー指定の変更", key: "name", input: `{"title":"T","name":"N"}`, want: "N"},
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

// TestConverter_ImplementsTitledConverter は、最適化された経路が
// 実際に選ばれる形になっていることを確認します。
func TestConverter_ImplementsTitledConverter(t *testing.T) {
	var titled ports.TitledConverter = New(mustParse(t, "test", `<h1>{{.title}}</h1>`))

	fragment, title, err := titled.ConvertWithTitle([]byte(`{"title":"タイトル"}`))
	require.NoError(t, err)
	assert.Equal(t, "タイトル", title)
	assert.Equal(t, "<h1>タイトル</h1>", string(fragment))
}
