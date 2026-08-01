package prompts

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Expand(t *testing.T) {
	t.Run("partialが差し込まれ、アクションはそのまま残る", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_output": "出力形式: JSON",
			"review":  "対象: {{.Target}}\n{{template \"_output\" .}}\n件数: {{.Count}}",
		})
		require.NoError(t, err)

		got, err := b.Expand("review")
		require.NoError(t, err)
		assert.Equal(t, "対象: {{.Target}}\n出力形式: JSON\n件数: {{.Count}}", got)
	})

	t.Run("入れ子のpartialも再帰的に展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_inner": "内側",
			"_outer": "外側[{{template \"_inner\" .}}]",
			"mode":   "本文 {{template \"_outer\" .}}",
		})
		require.NoError(t, err)

		got, err := b.Expand("mode")
		require.NoError(t, err)
		assert.Equal(t, "本文 外側[内側]", got)
	})

	t.Run("登録順に関係なく展開される", func(t *testing.T) {
		// 参照先が名前順で後ろにあっても解決される（展開順への依存がないこと）
		b, err := NewBuilder(map[string]string{
			"a_mode":  `{{template "z_part" .}}`,
			"z_part":  "後ろにある部品",
			"_unused": "未使用",
		})
		require.NoError(t, err)

		got, err := b.Expand("a_mode")
		require.NoError(t, err)
		assert.Equal(t, "後ろにある部品", got)
	})

	t.Run("引数なしの{{template}}も展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_p":   "部品",
			"mode": `[{{template "_p"}}]`,
		})
		require.NoError(t, err)

		got, err := b.Expand("mode")
		require.NoError(t, err)
		assert.Equal(t, "[部品]", got)
	})

	t.Run("Expandの結果はBuildの出力と整合する", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_greeting": "こんにちは、{{.Name}}さん",
			"main":      `{{template "_greeting" .}}！`,
		})
		require.NoError(t, err)

		expanded, err := b.Expand("main")
		require.NoError(t, err)
		assert.Equal(t, "こんにちは、{{.Name}}さん！", expanded)

		// 展開結果をそのまま実行すると Build と同じ出力になる
		data := struct{ Name string }{Name: "太郎"}
		built, err := b.Build("main", data)
		require.NoError(t, err)

		tpl := template.Must(template.New("x").Parse(expanded))
		var sb strings.Builder
		require.NoError(t, tpl.Execute(&sb, data))
		assert.Equal(t, built, sb.String())
	})

	t.Run("partialを持たないモードはそのまま返る", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{"plain": "ただの本文 {{.X}}"})
		require.NoError(t, err)

		got, err := b.Expand("plain")
		require.NoError(t, err)
		assert.Equal(t, "ただの本文 {{.X}}", got)
	})
}

func TestBuilder_Expand_ControlFlow(t *testing.T) {
	t.Run("ifの内側の参照も展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_p":   "部品",
			"mode": `{{if .Flag}}{{template "_p" .}}{{else}}なし{{end}}`,
		})
		require.NoError(t, err)

		got, err := b.Expand("mode")
		require.NoError(t, err)
		assert.Equal(t, "{{if .Flag}}部品{{else}}なし{{end}}", got)
	})

	t.Run("else節の内側の参照も展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_p":   "部品",
			"mode": `{{if .Flag}}あり{{else}}{{template "_p" .}}{{end}}`,
		})
		require.NoError(t, err)

		got, err := b.Expand("mode")
		require.NoError(t, err)
		assert.Equal(t, "{{if .Flag}}あり{{else}}部品{{end}}", got)
	})

	t.Run("rangeの内側の参照も展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_row":  "- {{.}}",
			"table": `{{range .Items}}{{template "_row" .}}{{end}}`,
		})
		require.NoError(t, err)

		got, err := b.Expand("table")
		require.NoError(t, err)
		assert.Equal(t, "{{range .Items}}- {{.}}{{end}}", got)
	})

	t.Run("withの内側の参照も展開される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_p":   "部品",
			"mode": `{{with .Sub}}{{template "_p" .}}{{end}}`,
		})
		require.NoError(t, err)

		got, err := b.Expand("mode")
		require.NoError(t, err)
		assert.Equal(t, "{{with .Sub}}部品{{end}}", got)
	})
}

func TestBuilder_Expand_Errors(t *testing.T) {
	t.Run("循環参照は検出される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_a":   `A{{template "_b" .}}`,
			"_b":   `B{{template "_a" .}}`,
			"mode": `{{template "_a" .}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCyclicTemplate)
	})

	t.Run("自己参照も検出される", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_self": `{{template "_self" .}}`,
			"mode":  `{{template "_self" .}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCyclicTemplate)
	})

	t.Run("引数付きの参照は展開せずエラーにする", func(t *testing.T) {
		// 展開すると部品内の . が別のものを指してしまうため
		b, err := NewBuilder(map[string]string{
			"_p":   "{{.Label}}",
			"mode": `{{template "_p" .Item}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotExpandable)
		assert.Contains(t, err.Error(), "_p")
	})

	t.Run("未知のモードはErrUnknownMode", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{"mode": "本文"})
		require.NoError(t, err)

		_, err = b.Expand("存在しない")
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("partialは直接展開できない", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_p":   "部品",
			"mode": "本文",
		})
		require.NoError(t, err)

		_, err = b.Expand("_p")
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("未定義の参照はErrUnknownMode", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"mode": `{{template "_missing" .}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("WithDefaultModeのフォールバックが効く", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"default": "既定の本文",
			"other":   "別の本文",
		}, WithDefaultMode("default"))
		require.NoError(t, err)

		got, err := b.Expand("")
		require.NoError(t, err)
		assert.Equal(t, "既定の本文", got)
	})
}

// TestBuilder_Expand_DoesNotMutate は、Expand が元の構文木を書き換えないことを確認します。
// 書き換えてしまうと以降の Build が壊れます。
func TestBuilder_Expand_DoesNotMutate(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"_p":   "部品({{.X}})",
		"mode": `前 {{template "_p" .}} 後`,
	})
	require.NoError(t, err)

	data := struct{ X string }{X: "値"}

	before, err := b.Build("mode", data)
	require.NoError(t, err)

	for range 3 {
		_, err := b.Expand("mode")
		require.NoError(t, err)
	}

	after, err := b.Build("mode", data)
	require.NoError(t, err)

	assert.Equal(t, before, after, "Expand の呼び出しが Build の結果を変えてはいけません")
	assert.Equal(t, "前 部品(値) 後", after)
}

// TestBuilder_Expand_NestedErrorPaths は、入れ子の内部で起きたエラーが
// 呼び出し元まで伝播することを確認します。
func TestBuilder_Expand_NestedErrorPaths(t *testing.T) {
	t.Run("ifの内側の未定義参照", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"mode": `{{if .Flag}}{{template "_missing" .}}{{end}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("else節の内側の未定義参照", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"mode": `{{if .Flag}}あり{{else}}{{template "_missing" .}}{{end}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		assert.ErrorIs(t, err, ErrUnknownMode)
	})

	t.Run("rangeの内側の引数付き参照", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_row": "{{.}}",
			"mode": `{{range .Items}}{{template "_row" .Sub}}{{end}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		assert.ErrorIs(t, err, ErrNotExpandable)
	})

	t.Run("partialの内側の未定義参照", func(t *testing.T) {
		b, err := NewBuilder(map[string]string{
			"_outer": `{{template "_missing" .}}`,
			"mode":   `{{template "_outer" .}}`,
		})
		require.NoError(t, err)

		_, err = b.Expand("mode")
		assert.ErrorIs(t, err, ErrUnknownMode)
	})
}

// TestBuilder_Expand_ArgumentForms は、展開できる参照とできない参照の
// 境界を網羅します。データコンテキストを変えるものだけを拒否します。
func TestBuilder_Expand_ArgumentForms(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		expandable bool
	}{
		{name: "引数なし", body: `{{template "_p"}}`, expandable: true},
		{name: "ドットのみ", body: `{{template "_p" .}}`, expandable: true},
		{name: "フィールド指定", body: `{{template "_p" .Item}}`, expandable: false},
		{name: "リテラル", body: `{{template "_p" "文字列"}}`, expandable: false},
		{name: "パイプライン", body: `{{template "_p" .A | printf "%v"}}`, expandable: false},
		{name: "変数宣言つき", body: `{{$x := .A}}{{template "_p" $x}}`, expandable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBuilder(map[string]string{
				"_p":   "部品",
				"mode": tt.body,
			})
			require.NoError(t, err)

			_, err = b.Expand("mode")
			if tt.expandable {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrNotExpandable)
		})
	}
}
