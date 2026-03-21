package converter

import (
	"testing"
)

func TestGoldmarkConverter_Convert(t *testing.T) {
	c := NewGoldmarkConverter()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "標準的なMarkdownの変換",
			input: "# Hello\nThis is **bold** text.",
			want:  "<h1>Hello</h1>\n<p>This is <strong>bold</strong> text.</p>\n",
		},
		{
			name:  "GFM拡張（テーブル）の変換",
			input: "| ID | Name |\n|---|---|\n| 1 | Alice |",
			want:  "<table>\n<thead>\n<tr>\n<th>ID</th>\n<th>Name</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>1</td>\n<td>Alice</td>\n</tr>\n</tbody>\n</table>\n",
		},
		{
			name:  "空入力",
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

func TestGoldmarkConverter_ExtractTitleFromMarkdown(t *testing.T) {
	c := NewGoldmarkConverter()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "H1見出しを抽出",
			input: "# プロジェクトタイトル\n本文が続く",
			want:  "プロジェクトタイトル",
		},
		{
			name:  "最初の見出しがH2の場合",
			input: "## サブタイトル\n本文",
			want:  "サブタイトル",
		},
		{
			name:  "インデントされた見出し",
			input: "   #  スペースありタイトル  ",
			want:  "スペースありタイトル",
		},
		{
			name:  "複数行ある場合の最初の見出し",
			input: "テキストのみの行\n# 最初の見出し\n## 二番目の見出し",
			want:  "最初の見出し",
		},
		{
			name:  "見出しがない場合",
			input: "これはただの本文です。\n- リスト1\n- リスト2",
			want:  "",
		},
		{
			name:  "シャープのみ（不正な形式）",
			input: "#NoSpaceTitle",
			want:  "",
		},
		{
			name:  "最大レベルのH6見出し",
			input: "###### 最小の見出し",
			want:  "最小の見出し",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.ExtractTitleFromMarkdown([]byte(tt.input)); got != tt.want {
				t.Errorf("ExtractTitleFromMarkdown() = %v, want %v", got, tt.want)
			}
		})
	}
}
