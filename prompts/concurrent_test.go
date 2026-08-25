package prompts

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder_Concurrent は、構築後の Builder を複数の goroutine から
// 同時に使えることを確認します。
//
// 利用側は HTTP ハンドラから Build を呼びます（起動時に一度構築して使い回す形）。
// 保証を書いただけで固定しないと、内部に状態を持つ変更が入ったときに
// 気付けるのが本番のデータ競合になります。-race での実行に意味があるテストです。
func TestBuilder_Concurrent(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"greet":    `こんにちは、{{.Name}}さん。{{template "_sign" .}}`,
		"farewell": `さようなら、{{.Name}}さん。{{template "_sign" .}}`,
		"_sign":    `（{{.Name}} 宛）`,
	})
	require.NoError(t, err)

	const goroutines = 32
	const iterations = 20

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			name := fmt.Sprintf("利用者%d", i)
			mode := "greet"
			want := fmt.Sprintf("こんにちは、%sさん。（%s 宛）", name, name)
			if i%2 == 1 {
				mode = "farewell"
				want = fmt.Sprintf("さようなら、%sさん。（%s 宛）", name, name)
			}

			for range iterations {
				got, err := b.Build(mode, struct{ Name string }{Name: name})
				assert.NoError(t, err)
				assert.Equal(t, want, got, "並行実行で出力が混ざっています")
			}
		}(i)
	}
	wg.Wait()
}

// TestBuilder_ConcurrentReaders は、参照系のメソッドも並行に呼べることを確認します。
func TestBuilder_ConcurrentReaders(t *testing.T) {
	b, err := NewBuilder(map[string]string{
		"greet": "こんにちは",
		"_sign": "（署名）",
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {

			assert.Equal(t, []string{"greet"}, b.Modes())
			assert.True(t, b.Has("greet"))
			assert.False(t, b.Has("_sign"))
			assert.Empty(t, b.FrontMatter("greet"))
			assert.Empty(t, b.FrontMatters())

			expanded, err := b.Expand("greet")
			assert.NoError(t, err)
			assert.Equal(t, "こんにちは", expanded)
		})
	}
	wg.Wait()
}

// TestBuilder_ModesIsACopy は、Modes の戻り値を書き換えても
// Builder の状態が壊れないことを確認します（並行利用の前提です）。
func TestBuilder_ModesIsACopy(t *testing.T) {
	b, err := NewBuilder(map[string]string{"greet": "こんにちは", "bye": "さようなら"})
	require.NoError(t, err)

	modes := b.Modes()
	modes[0] = "書き換え"

	assert.Equal(t, []string{"bye", "greet"}, b.Modes())
}
