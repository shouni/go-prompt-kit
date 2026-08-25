// Package jsondoc は、任意の構造を持つJSON入力を、呼び出し側が指定した
// html/template を通じてHTMLフラグメントへ変換する ports.Converter 実装を提供します。
// markdown.Converter がMarkdownの中身に関知しないのと同様、JSONのスキーマや
// テンプレートの中身には関知しません — それらは呼び出し側の責務です。
//
// 数値は json.Number（入力に書かれた字面のままの文字列）としてテンプレートへ渡します。
// float64 を経由すると桁の大きい整数が指数表記になり、末尾が 0 の小数も丸められて、
// 文書に載る数が入力と食い違うためです。
package jsondoc

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/shouni/go-prompt-kit/htmldoc/ports"
)

var (
	_ ports.Converter       = (*Converter)(nil)
	_ ports.TitledConverter = (*Converter)(nil)
)

const defaultTitleKey = "title"

// Converter は JSON 用の ports.Converter 実装で、
// JSON入力を呼び出し側のテンプレートでHTMLフラグメントへ変換します。
type Converter struct {
	tpl      *template.Template
	titleKey string
}

// New は、HTMLフラグメントの生成に使うテンプレートを受け取り Converter を構築します。
func New(tpl *template.Template, opts ...Option) *Converter {
	c := &Converter{
		tpl:      tpl,
		titleKey: defaultTitleKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert は、JSON入力をデコードし、テンプレートに渡してHTMLフラグメントを生成します。
func (c *Converter) Convert(input []byte) ([]byte, error) {
	fragment, _, err := c.convert(input, false)

	return fragment, err
}

// ConvertWithTitle は、入力を1回だけデコードしてHTMLフラグメントとタイトルを返します
// （ports.TitledConverter の実装です）。
// Convert と ExtractTitle を続けて呼ぶのと結果は同じですが、デコードは1回で済みます。
func (c *Converter) ConvertWithTitle(input []byte) ([]byte, string, error) {
	return c.convert(input, true)
}

// convert は、デコード・レンダリング・タイトル取得をまとめた実体です。
func (c *Converter) convert(input []byte, withTitle bool) ([]byte, string, error) {
	if len(input) == 0 {
		return nil, "", nil
	}

	data, err := decode(input)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := c.tpl.Execute(&buf, data); err != nil {
		return nil, "", fmt.Errorf("HTMLフラグメントのレンダリングに失敗しました: %w", err)
	}

	var title string
	if withTitle {
		title = c.titleOf(data)
	}

	return buf.Bytes(), title, nil
}

// decode は、JSON入力を any へ読み取ります。
//
// 数値は json.Number（元の字面のままの文字列）として保持します。既定の float64 へ
// 変換すると、桁の大きい整数が "1.234567890123e+12" という指数表記になり、
// 末尾が 0 の小数（0.30）も 0.3 に丸められます。テンプレートはそれをそのまま
// 出力するため、入力に書かれた数と表示される数が食い違います。
// 文書として書き出すのがこのパッケージの目的なので、字面を優先します。
//
// テンプレート側で数値として比較・計算したい場合は、json.Number を受け取って
// 変換する関数を Funcs で登録してください。
func decode(input []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()

	var data any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON入力のパースに失敗しました: %w", err)
	}

	// json.Unmarshal と違い Decoder は値の後ろを読み飛ばすため、
	// 入力が1つの値で終わっていることを確かめます。
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("JSON入力のパースに失敗しました: 値の後ろに余分なデータがあります")
	}

	return data, nil
}

// titleOf は、デコード済みの値のトップレベルから titleKey の文字列値を取り出します。
func (c *Converter) titleOf(data any) string {
	object, ok := data.(map[string]any)
	if !ok {
		return ""
	}

	title, _ := object[c.titleKey].(string)

	return title
}

// ExtractTitle は、JSON入力のトップレベルオブジェクトから
// titleKey（既定は "title"）に対応する文字列値を抽出します。
//
// 入力全体をデコードせず、目的のキーが見つかった時点で読み取りを打ち切ります。
// そのため、キーより後ろが壊れた入力でもタイトルは取り出せます（その入力は
// Convert 側で改めてエラーになります）。
// 変換と同時にタイトルが必要な場合は、入力を2度読まない ConvertWithTitle を使ってください。
func (c *Converter) ExtractTitle(input []byte) string {
	dec := jsontext.NewDecoder(bytes.NewReader(input))
	if dec.PeekKind() != jsontext.KindBeginObject {
		return ""
	}
	if _, err := dec.ReadToken(); err != nil {
		return ""
	}

	for dec.PeekKind() != jsontext.KindEndObject {
		name, err := dec.ReadToken()
		if err != nil {
			return ""
		}

		if name.String() != c.titleKey {
			if err := dec.SkipValue(); err != nil {
				return ""
			}
			continue
		}

		// 値が文字列でなければタイトルとしては扱いません。
		if dec.PeekKind() != jsontext.KindString {
			return ""
		}
		value, err := dec.ReadToken()
		if err != nil {
			return ""
		}

		return value.String()
	}

	return ""
}
