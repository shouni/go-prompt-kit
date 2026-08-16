package frontmatter

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ErrNoUnmarshalFunc は、解析関数が渡されなかった場合に返されます。
var ErrNoUnmarshalFunc = errors.New("front matter の解析関数が指定されていません")

// UnmarshalFunc は、front matter を構造体へ読み取る関数です。
// yaml.Unmarshal や json.Unmarshal をそのまま渡せます。
//
// このパッケージがYAMLライブラリを直接参照せず呼び出し側から受け取るのは、
// ライブラリを利用側が選べるようにするためです。ライブラリを固定すると、
// 乗り換えのたびにこのモジュールのリリースと足並みを揃える必要が生じ、
// 移行の途中では1つのバイナリに2つの実装が載ります。
type UnmarshalFunc func(data []byte, v any) error

// Decode は、Split が返した front matter を v へ読み取ります。
// front matter が空の場合は何もせず nil を返すため、メタデータを持たない
// プロンプトもそのまま扱えます。
func Decode(front string, v any, unmarshal UnmarshalFunc) error {
	// データの有無にかかわらず先に検査します。空のエントリだけを渡したときに
	// 配線の誤りが表面化しないのを避けるためです。
	if unmarshal == nil {
		return ErrNoUnmarshalFunc
	}
	if strings.TrimSpace(front) == "" {
		return nil
	}

	if err := unmarshal([]byte(front), v); err != nil {
		return fmt.Errorf("front matter の解析に失敗しました: %w", err)
	}

	return nil
}

// DecodeMap は、SplitMap が返した front matter をまとめて T へ読み取ります。
// キーの集合は入力と同じで、front matter を持たないエントリは T のゼロ値になります。
//
// 解析に失敗したエントリがあれば、そのキー名を添えてエラーを返します
// （書き損じたメタデータが空欄として素通りするのを防ぐためです）。
// エントリごとに扱いを変えたい場合は、Split と Decode を直接使ってください。
func DecodeMap[T any](fronts map[string]string, unmarshal UnmarshalFunc) (map[string]T, error) {
	if unmarshal == nil {
		return nil, ErrNoUnmarshalFunc
	}

	out := make(map[string]T, len(fronts))
	// 複数のエントリが壊れている場合でも報告するキーが変わらないよう、名前順に処理します。
	for _, name := range slices.Sorted(maps.Keys(fronts)) {
		var v T
		if err := Decode(fronts[name], &v, unmarshal); err != nil {
			return nil, fmt.Errorf("'%s': %w", name, err)
		}
		out[name] = v
	}

	return out, nil
}
