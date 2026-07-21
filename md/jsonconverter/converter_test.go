package jsonconverter

import (
	"html/template"
	"testing"
)

func mustParse(t *testing.T, name, body string) *template.Template {
	t.Helper()
	tpl, err := template.New(name).Parse(body)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	return tpl
}

func TestJSONConverter_Convert(t *testing.T) {
	tpl := mustParse(t, "test", `<h1>{{.title}}</h1><p>{{.summary}}</p>`)
	c := New(tpl)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "任意のJSON構造をテンプレートに渡す",
			input: `{"title":"Title","summary":"Summary"}`,
			want:  "<h1>Title</h1><p>Summary</p>",
		},
		{
			name:  "空入力",
			input: "",
			want:  "",
		},
		{
			name:    "不正なJSON",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Convert([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Convert() got = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestJSONConverter_ExtractTitleFromMarkdown(t *testing.T) {
	tpl := mustParse(t, "test", `{{.title}}`)

	tests := []struct {
		name  string
		opts  []Option
		input string
		want  string
	}{
		{
			name:  "デフォルトキー(title)から抽出",
			input: `{"title":"プロジェクトタイトル"}`,
			want:  "プロジェクトタイトル",
		},
		{
			name:  "キーが存在しない",
			input: `{"summary":"本文"}`,
			want:  "",
		},
		{
			name:  "不正なJSON",
			input: `not json`,
			want:  "",
		},
		{
			name:  "カスタムキーから抽出",
			opts:  []Option{WithTitleKey("heading")},
			input: `{"heading":"カスタムタイトル"}`,
			want:  "カスタムタイトル",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tpl, tt.opts...)
			if got := c.ExtractTitleFromMarkdown([]byte(tt.input)); got != tt.want {
				t.Errorf("ExtractTitleFromMarkdown() = %v, want %v", got, tt.want)
			}
		})
	}
}
