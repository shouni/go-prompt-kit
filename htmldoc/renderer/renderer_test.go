package renderer

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// Helper that creates a mock Renderer for tests. It builds the struct directly, assuming the real
// assets (embed.FS) may not be accessible in the test environment.
func newTestRenderer() *Renderer {
	// A simple HTML template for tests
	tplText := `<!DOCTYPE html><html lang="{{.Lang}}"><head><title>{{.Title}}</title><style>{{.Style}}</style></head><body>{{.Content}}</body></html>`
	tpl := template.Must(template.New("template.html").Parse(tplText))

	return &Renderer{
		tpl: tpl,
		css: template.CSS("body { color: red; }"),
	}
}

func TestRenderer_Render(t *testing.T) {
	r := newTestRenderer()

	tests := []struct {
		name     string
		bodyHTML []byte
		lang     string
		title    string
		contains []string // strings the output should contain
	}{
		{
			name:     "正常系: 標準的なレンダリング",
			bodyHTML: []byte("<p>Hello World</p>"),
			lang:     "ja",
			title:    "テストタイトル",
			contains: []string{
				"lang=\"ja\"",
				"<title>テストタイトル</title>",
				"body { color: red; }",
				"<p>Hello World</p>",
			},
		},
		{
			name:     "空のコンテンツ",
			bodyHTML: []byte(""),
			lang:     "en",
			title:    "Empty Page",
			contains: []string{
				"lang=\"en\"",
				"<title>Empty Page</title>",
			},
		},
		{
			name:     "HTMLエスケープがされないことの確認",
			bodyHTML: []byte("<script>alert('hi')</script>"),
			lang:     "ja",
			title:    "XSS Check",
			contains: []string{
				"<script>alert('hi')</script>", // not escaped, since it is a template.HTML
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := r.Render(&buf, tt.bodyHTML, tt.lang, tt.title)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			got := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Render() output does not contain %q\nGot: %s", want, got)
				}
			}
		})
	}
}
