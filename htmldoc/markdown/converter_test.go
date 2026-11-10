package markdown

import (
	"testing"
)

func TestConverter_Convert(t *testing.T) {
	c := New()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "standard Markdown",
			input: "# Hello\nThis is **bold** text.",
			want:  "<h1>Hello</h1>\n<p>This is <strong>bold</strong> text.</p>\n",
		},
		{
			name:  "GFM extension (table)",
			input: "| ID | Name |\n|---|---|\n| 1 | Alice |",
			want:  "<table>\n<thead>\n<tr>\n<th>ID</th>\n<th>Name</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>1</td>\n<td>Alice</td>\n</tr>\n</tbody>\n</table>\n",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
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
				t.Errorf("Convert() got = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestConverter_ExtractTitle(t *testing.T) {
	c := New()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts an H1 heading",
			input: "# プロジェクトタイトル\n本文が続く",
			want:  "プロジェクトタイトル",
		},
		{
			name:  "first heading is H2",
			input: "## サブタイトル\n本文",
			want:  "サブタイトル",
		},
		{
			name:  "indented heading",
			input: "   #  スペースありタイトル  ",
			want:  "スペースありタイトル",
		},
		{
			name:  "first heading of several lines",
			input: "テキストのみの行\n# 最初の見出し\n## 二番目の見出し",
			want:  "最初の見出し",
		},
		{
			name:  "no heading",
			input: "これはただの本文です。\n- リスト1\n- リスト2",
			want:  "",
		},
		{
			name:  "hash only (malformed)",
			input: "#NoSpaceTitle",
			want:  "",
		},
		{
			name:  "H6 heading at the max level",
			input: "###### 最小の見出し",
			want:  "最小の見出し",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.ExtractTitle([]byte(tt.input)); got != tt.want {
				t.Errorf("ExtractTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
