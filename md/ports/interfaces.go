package ports

import (
	"bytes"
	"io"
)

// Runner は Markdown 変換処理を実行するコアサービスを抽象化します。
type Runner interface {
	Run(title string, markdown []byte) (*bytes.Buffer, error)
}

// Converter は Markdown を HTML フラグメントに変換するサービスのインターフェースです。
type Converter interface {
	Convert(input []byte) ([]byte, error)
	ExtractTitleFromMarkdown(input []byte) string
}

// Renderer は、汎用的なMarkdownからHTMLドキュメントを生成するインターフェースです。
type Renderer interface {
	Render(writer io.Writer, bodyHTML []byte, lang, title string) error
}
