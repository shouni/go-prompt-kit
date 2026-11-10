package prompts

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder_Concurrent confirms that a built Builder can be used from several goroutines at once.
//
// Consumers call Build from HTTP handlers (built once at startup and reused). A guarantee that is
// only written down and not pinned means a change that adds internal state would first be noticed
// as a data race in production. The test means something when run with -race.
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
				assert.Equal(t, want, got, "outputs got mixed up under concurrent execution")
			}
		}(i)
	}
	wg.Wait()
}

// TestBuilder_ConcurrentReaders confirms that the read-only methods can be called concurrently too.
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

			// It walks the parse trees as Expand does, so not rewriting the trees is what matters
			// here.
			fields, err := b.Fields("greet")
			assert.NoError(t, err)
			assert.Empty(t, fields)
		})
	}
	wg.Wait()
}

// TestBuilder_ModesIsACopy confirms that modifying the slice Modes returns does not corrupt the
// Builder's state (a precondition for concurrent use).
func TestBuilder_ModesIsACopy(t *testing.T) {
	b, err := NewBuilder(map[string]string{"greet": "こんにちは", "bye": "さようなら"})
	require.NoError(t, err)

	modes := b.Modes()
	modes[0] = "書き換え"

	assert.Equal(t, []string{"bye", "greet"}, b.Modes())
}
