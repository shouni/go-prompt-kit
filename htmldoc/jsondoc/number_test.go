package jsondoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_Convert_NumberFidelity は、数値が入力に書かれた字面のまま
// 出力されることを確認します。float64 を経由すると桁の大きい整数が指数表記になり、
// 末尾が 0 の小数も丸められて、文書に載る数が入力と食い違います。
func TestConverter_Convert_NumberFidelity(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "桁の大きい整数",
			input: `{"value":1234567890123}`,
			want:  "1234567890123",
		},
		{
			name:  "末尾が0の小数",
			input: `{"value":0.30}`,
			want:  "0.30",
		},
		{
			name:  "float64で表現できない精度",
			input: `{"value":12345678901234567890}`,
			want:  "12345678901234567890",
		},
		{
			name:  "指数表記はそのまま",
			input: `{"value":1e3}`,
			want:  "1e3",
		},
		{
			name:  "小さい整数",
			input: `{"value":42}`,
			want:  "42",
		},
	}

	c := New(mustParse(t, "num", `{{.value}}`))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Convert([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

// TestConverter_Convert_TrailingData は、1つの値の後ろに余分なデータがある入力を
// 拒むことを確認します。json.Unmarshal は拒みますが、json.Decoder は読み飛ばすため、
// 明示的に確かめないと壊れた入力の前半だけが黙って描画されます。
func TestConverter_Convert_TrailingData(t *testing.T) {
	c := New(mustParse(t, "t", `{{.a}}`))

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "末尾の空白は許容", input: "{\"a\":1}\n  "},
		{name: "2つ目の値", input: `{"a":1} {"b":2}`, wantErr: true},
		{name: "壊れた末尾", input: `{"a":1}]`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Convert([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
