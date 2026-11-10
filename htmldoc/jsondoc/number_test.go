package jsondoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverter_Convert_NumberFidelity confirms that numbers are output exactly as written in the
// input. Going through float64 would put large integers into exponent notation and round decimals
// ending in 0, so the numbers in the document would disagree with the input.
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

// TestConverter_Convert_TrailingData confirms that input with extra data after one value is
// rejected. json.Unmarshal rejects it but json.Decoder skips it, so without an explicit check only
// the first half of a broken input would be rendered silently.
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
