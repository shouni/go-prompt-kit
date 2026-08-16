// Package ports は、HTMLドキュメント生成が依存するインターフェース（ポート）を定義します。
// 実装から独立した専用パッケージに置くことで、実装側（markdown / jsondoc / renderer）と
// 利用側（htmldoc）の双方が循環参照なしにインターフェースを参照できます。
package ports

import (
	"io"
)

// Runner は、入力を完全なHTMLドキュメントとして書き出すコアサービスを抽象化します。
// 入力形式は注入される Converter が決めるため、この抽象は形式に依存しません。
type Runner interface {
	Run(writer io.Writer, title string, input []byte) error
}

// Converter は、入力をHTMLフラグメント（ドキュメント全体ではなく本文部分）へ
// 変換するサービスのインターフェースです。
type Converter interface {
	// Convert は入力をHTMLフラグメントへ変換します。
	Convert(input []byte) ([]byte, error)
	// ExtractTitle は入力からドキュメントのタイトルを抽出します。
	// 抽出できない場合は空文字を返します。
	ExtractTitle(input []byte) string
}

// Renderer は、HTMLフラグメントを完全なHTMLドキュメントへ組み立てるインターフェースです。
type Renderer interface {
	Render(writer io.Writer, bodyHTML []byte, lang, title string) error
}
