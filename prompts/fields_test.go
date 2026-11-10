package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder_Fields checks the listing of the fields a mode requires.
func TestBuilder_Fields(t *testing.T) {
	tests := []struct {
		name      string
		templates map[string]string
		mode      string
		want      []string
	}{
		{
			name:      "simple reference",
			templates: map[string]string{"m": `{{.Name}}さん、{{.Message}}`},
			mode:      "m",
			want:      []string{"Message", "Name"},
		},
		{
			name:      "nested fields joined with .",
			templates: map[string]string{"m": `{{.Source.Title}} / {{.Source.Author.Name}}`},
			mode:      "m",
			want:      []string{"Source.Author.Name", "Source.Title"},
		},
		{
			name:      "deduplicates",
			templates: map[string]string{"m": `{{.Name}}{{.Name}}{{.Name}}`},
			mode:      "m",
			want:      []string{"Name"},
		},
		{
			name:      "no references",
			templates: map[string]string{"m": `固定の指示文です。`},
			mode:      "m",
			want:      nil,
		},
		{
			name:      "bare dot is not recorded",
			templates: map[string]string{"m": `{{.}}`},
			mode:      "m",
			want:      nil,
		},
		{
			name:      "follows if bodies since if keeps .",
			templates: map[string]string{"m": `{{if .Detailed}}{{.LongForm}}{{else}}{{.ShortForm}}{{end}}`},
			mode:      "m",
			want:      []string{"Detailed", "LongForm", "ShortForm"},
		},
		{
			name:      "follows into partials",
			templates: map[string]string{"m": `{{.Task}}{{template "_sign" .}}`, "_sign": `-- {{.Author}}`},
			mode:      "m",
			want:      []string{"Author", "Task"},
		},
		{
			name:      "function arguments",
			templates: map[string]string{"m": `{{printf "%s/%s" .A .B}}`},
			mode:      "m",
			want:      []string{"A", "B"},
		},
		{
			name:      "nested pipeline",
			templates: map[string]string{"m": `{{printf "%s" (printf "%s" .Inner)}}`},
			mode:      "m",
			want:      []string{"Inner"},
		},
		{
			name:      "variable assignment source",
			templates: map[string]string{"m": `{{$name := .Name}}{{$name}}`},
			mode:      "m",
			want:      []string{"Name"},
		},
		{
			name:      "parenthesized reference",
			templates: map[string]string{"m": `{{(.User).Name}}`},
			mode:      "m",
			want:      []string{"User.Name"},
		},
		{
			name:      "falls back to the default mode",
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

// TestBuilder_Fields_Rebound pins the handling of spans where range and with rebind ".". Fields
// inside them have no fixed position in data and are not listed, but the target expression, the
// else clause and $-rooted references have a fixed position and are listed.
func TestBuilder_Fields_Rebound(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "lists the range target but not its body",
			template: `{{range .Items}}{{.Title}}{{end}}`,
			want:     []string{"Items"},
		},
		{
			name:     "range else keeps .",
			template: `{{range .Items}}{{.Title}}{{else}}{{.EmptyMessage}}{{end}}`,
			want:     []string{"EmptyMessage", "Items"},
		},
		{
			name:     "$ inside a range body is resolved",
			template: `{{range .Items}}{{$.Language}}: {{.Title}}{{end}}`,
			want:     []string{"Items", "Language"},
		},
		{
			name:     "with behaves the same",
			template: `{{with .Source}}{{.Title}}{{end}}`,
			want:     []string{"Source"},
		},
		{
			name:     "if inside a range body keeps the rebound .",
			template: `{{range .Items}}{{if .Done}}{{$.Mark}}{{end}}{{end}}`,
			want:     []string{"Items", "Mark"},
		},
		{
			name:     "nested range",
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

// TestBuilder_Fields_Errors checks the cases refused for the same reasons as Expand.
func TestBuilder_Fields_Errors(t *testing.T) {
	tests := []struct {
		name      string
		templates map[string]string
		mode      string
		wantErr   error
	}{
		{
			name:      "unregistered mode",
			templates: map[string]string{"m": `{{.Name}}`},
			mode:      "unknown",
			wantErr:   ErrUnknownMode,
		},
		{
			name:      "partial is not accepted",
			templates: map[string]string{"m": `x`, "_p": `{{.Name}}`},
			mode:      "_p",
			wantErr:   ErrUnknownMode,
		},
		{
			name:      "reference that changes the data context",
			templates: map[string]string{"m": `{{template "_p" .Foo}}`, "_p": `{{.Name}}`},
			mode:      "m",
			wantErr:   ErrNotExpandable,
		},
		{
			name:      "cycle",
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

// TestBuilder_Fields_MatchesBuild confirms that Build succeeds with data holding exactly the listed
// fields. Being the counterpart of missingkey=error is why Fields exists, so a change to only one
// side would make it pointless.
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

	// Missing even one makes Build fail (the flip side of the returned set being exactly right).
	delete(data, "Task")
	_, err = b.Build("m", data)
	assert.Error(t, err)
}
