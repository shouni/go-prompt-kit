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

// TitledConverter は、変換とタイトル抽出を1回の解析でまとめて行える Converter です。
//
// Converter だけを実装していれば Document は動きますが、その場合タイトルを
// 自動抽出するたびに Convert と ExtractTitle が同じ入力をそれぞれ解析するため、
// 入力を2回解析することになります。これを実装していると Document はこちらを
// 優先して呼び、解析を1回で済ませます。
//
// 同梱の markdown.Converter と jsondoc.Converter はどちらも実装しています。
type TitledConverter interface {
	Converter

	// ConvertWithTitle は、入力を1回だけ解析してHTMLフラグメントとタイトルを返します。
	// タイトルを抽出できない場合、title は空文字になります（エラーではありません）。
	//
	// 返す値は Convert と ExtractTitle をそれぞれ呼んだ場合と一致していなければなりません。
	ConvertWithTitle(input []byte) (fragment []byte, title string, err error)
}

// Renderer は、HTMLフラグメントを完全なHTMLドキュメントへ組み立てるインターフェースです。
type Renderer interface {
	Render(writer io.Writer, bodyHTML []byte, lang, title string) error
}
