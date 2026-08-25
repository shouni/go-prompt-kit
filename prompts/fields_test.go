package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder_Fields は、モードが要求するフィールドの列挙を確認します。
func TestBuilder_Fields(t *testing.T) {
	tests := []struct {
		name      string
		templates map[string]string
		mode      string
		want      []string
	}{
		{
			name:      "単純な参照",
			templates: map[string]string{"m": `{{.Name}}さん、{{.Message}}`},
			mode:      "m",
			want:      []string{"Message", "Name"},
		},
		{
			name:      "ネストは.で繋ぐ",
			templates: map[string]string{"m": `{{.Source.Title}} / {{.Source.Author.Name}}`},
			mode:      "m",
			want:      []string{"Source.Author.Name", "Source.Title"},
		},
		{
			name:      "重複は1つにまとめる",
			templates: map[string]string{"m": `{{.Name}}{{.Name}}{{.Name}}`},
			mode:      "m",
			want:      []string{"Name"},
		},
		{
			name:      "参照なし",
			templates: map[string]string{"m": `固定の指示文です。`},
			mode:      "m",
			want:      nil,
		},
		{
			name:      "ドット単体は記録しない",
			templates: map[string]string{"m": `{{.}}`},
			mode:      "m",
			want:      nil,
		},
		{
			name:      "if は . を変えないので本体も辿る",
			templates: map[string]string{"m": `{{if .Detailed}}{{.LongForm}}{{else}}{{.ShortForm}}{{end}}`},
			mode:      "m",
			want:      []string{"Detailed", "LongForm", "ShortForm"},
		},
		{
			name:      "partial の中身も辿る",
			templates: map[string]string{"m": `{{.Task}}{{template "_sign" .}}`, "_sign": `-- {{.Author}}`},
			mode:      "m",
			want:      []string{"Author", "Task"},
		},
		{
			name:      "関数の引数",
			templates: map[string]string{"m": `{{printf "%s/%s" .A .B}}`},
			mode:      "m",
			want:      []string{"A", "B"},
		},
		{
			name:      "入れ子のパイプライン",
			templates: map[string]string{"m": `{{printf "%s" (printf "%s" .Inner)}}`},
			mode:      "m",
			want:      []string{"Inner"},
		},
		{
			name:      "変数への代入元",
			templates: map[string]string{"m": `{{$name := .Name}}{{$name}}`},
			mode:      "m",
			want:      []string{"Name"},
		},
		{
			name:      "括弧を挟んだ参照",
			templates: map[string]string{"m": `{{(.User).Name}}`},
			mode:      "m",
			want:      []string{"User.Name"},
		},
		{
			name:      "既定モードへのフォールバック",
			templates: map[string]string{"fallback": `{{.Name}}`},
			mode:      "未登録",
			want:      []string{"Name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if _, ok := tt.templates["fallback"]; ok {
				opts = append(opts, WithDefaultMode("fallback"))
			}

			b, err := NewBuilder(tt.templates, opts...)
			require.NoError(t, err)

			got, err := b.Fields(tt.mode)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuilder_Fields_Rebound は、range と with で "." が差し替わる区間の扱いを固定します。
// 内側のフィールドは data からの位置が決まらないため列挙しませんが、
// 対象の式・else 節・$ 起点の参照は位置が確定するので列挙します。
func TestBuilder_Fields_Rebound(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "rangeの対象は列挙し本体は列挙しない",
			template: `{{range .Items}}{{.Title}}{{end}}`,
			want:     []string{"Items"},
		},
		{
			name:     "rangeのelse節は . が変わらない",
			template: `{{range .Items}}{{.Title}}{{else}}{{.EmptyMessage}}{{end}}`,
			want:     []string{"EmptyMessage", "Items"},
		},
		{
			name:     "range本体の $ 起点は位置が決まる",
			template: `{{range .Items}}{{$.Language}}: {{.Title}}{{end}}`,
			want:     []string{"Items", "Language"},
		},
		{
			name:     "withも同じ扱い",
			template: `{{with .Source}}{{.Title}}{{end}}`,
			want:     []string{"Source"},
		},
		{
			name:     "range本体の中のifも差し替わったまま",
			template: `{{range .Items}}{{if .Done}}{{$.Mark}}{{end}}{{end}}`,
			want:     []string{"Items", "Mark"},
		},
		{
			name:     "入れ子のrange",
			template: `{{range .Groups}}{{range .Items}}{{$.Prefix}}{{end}}{{end}}`,
			want:     []string{"Groups", "Prefix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBuilder(map[string]string{"m": tt.template})
			require.NoError(t, err)

			got, err := b.Fields("m")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBuilder_Fields_Errors は、Expand と同じ理由で断る場合を確認します。
func TestBuilder_Fields_Errors(t *testing.T) {
	tests := []struct {
		name      string
		templates map[string]string
		mode      string
		wantErr   error
	}{
		{
			name:      "未登録のモード",
			templates: map[string]string{"m": `{{.Name}}`},
			mode:      "unknown",
			wantErr:   ErrUnknownMode,
		},
		{
			name:      "partial は指定できない",
			templates: map[string]string{"m": `x`, "_p": `{{.Name}}`},
			mode:      "_p",
			wantErr:   ErrUnknownMode,
		},
		{
			name:      "データコンテキストを変える参照",
			templates: map[string]string{"m": `{{template "_p" .Foo}}`, "_p": `{{.Name}}`},
			mode:      "m",
			wantErr:   ErrNotExpandable,
		},
		{
			name:      "循環参照",
			templates: map[string]string{"m": `{{template "_a" .}}`, "_a": `{{template "_b" .}}`, "_b": `{{template "_a" .}}`},
			mode:      "m",
			wantErr:   ErrCyclicTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBuilder(tt.templates)
			require.NoError(t, err)

			_, err = b.Fields(tt.mode)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestBuilder_Fields_MatchesBuild は、列挙されたフィールドを揃えたデータなら
// Build が通ることを確認します。missingkey=error との対であることが Fields の
// 存在理由なので、片方だけ変わると意味を失います。
func TestBuilder_Fields_MatchesBuild(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"m":     `{{.Language}}で{{.Task}}を行ってください。{{template "_sign" .}}`,
		"_sign": `-- {{.Author}}`,
	})
	require.NoError(t, err)

	fields, err := b.Fields("m")
	require.NoError(t, err)
	require.Equal(t, []string{"Author", "Language", "Task"}, fields)

	data := make(map[string]any, len(fields))
	for _, name := range fields {
		data[name] = "値"
	}

	got, err := b.Build("m", data)
	require.NoError(t, err)
	assert.Equal(t, "値で値を行ってください。-- 値", got)

	// 1つでも欠ければ Build は失敗します（Fields が返した集合が過不足ないことの裏です）。
	delete(data, "Task")
	_, err = b.Build("m", data)
	assert.Error(t, err)
}
