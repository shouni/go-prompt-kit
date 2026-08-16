package prompts

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBuilder_RejectsLoadOnlyOptions は、読み込み時のみ有効なオプションを
// NewBuilder が受け付けないことを確認します。
//
// 黙って無視すると、絞り込んだつもりのまま全エントリが登録され、
// しかもコンパイルも実行も通ってしまいます。
func TestNewBuilder_RejectsLoadOnlyOptions(t *testing.T) {
	templates := map[string]string{"greet": "こんにちは"}

	tests := []struct {
		name     string
		option   Option
		wantName string
	}{
		{name: "WithPrefix", option: WithPrefix("prompt_"), wantName: "WithPrefix"},
		{name: "WithRecursive", option: WithRecursive(), wantName: "WithRecursive"},
		{name: "WithExtensions", option: WithExtensions(".md"), wantName: "WithExtensions"},
		{name: "WithFrontMatter", option: WithFrontMatter(), wantName: "WithFrontMatter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBuilder(templates, tt.option)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrLoadOnlyOption)
			assert.Contains(t, err.Error(), tt.wantName, "どのオプションが問題かを示すべきです")
		})
	}

	t.Run("複数指定した場合はすべて報告する", func(t *testing.T) {
		_, err := NewBuilder(templates, WithRecursive(), WithExtensions(".md"))

		require.ErrorIs(t, err, ErrLoadOnlyOption)
		assert.Contains(t, err.Error(), "WithRecursive")
		assert.Contains(t, err.Error(), "WithExtensions")
	})

	t.Run("構築時のオプションはそのまま使える", func(t *testing.T) {
		_, err := NewBuilder(templates,
			WithDefaultMode("greet"),
			WithPartialPrefix("#"),
		)

		require.NoError(t, err)
	})

	// LoadFS は読み込み専用オプションを自身で消費するため、検査に引っかかりません。
	t.Run("LoadFS では受け付ける", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"prompts/prompt_greet.md": &fstest.MapFile{Data: []byte("こんにちは")},
			"prompts/skip.txt":        &fstest.MapFile{Data: []byte("対象外")},
		}

		b, err := LoadFS(mockFS, "prompts", WithPrefix("prompt_"), WithExtensions(".md"))

		require.NoError(t, err)
		assert.Equal(t, []string{"greet"}, b.Modes())
	})
}

func TestIsPartial(t *testing.T) {
	tests := []struct {
		name   string
		entry  string
		prefix string
		want   bool
	}{
		{name: "既定の接頭辞", entry: "_output", prefix: DefaultPartialPrefix, want: true},
		{name: "モード名", entry: "review", prefix: DefaultPartialPrefix, want: false},
		{name: "再帰読み込みのパス形式", entry: "en/_output", prefix: DefaultPartialPrefix, want: true},
		{name: "パス形式のモード名", entry: "en/review", prefix: DefaultPartialPrefix, want: false},
		{name: "ディレクトリ名だけが接頭辞に一致", entry: "_shared/review", prefix: DefaultPartialPrefix, want: false},
		{name: "独自の接頭辞", entry: "compose_structure", prefix: "compose_", want: true},
		{name: "接頭辞が空なら判定しない", entry: "_output", prefix: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPartial(tt.entry, tt.prefix))
		})
	}
}

// TestIsPartial_MatchesBuilder は、公開した IsPartial の判定が
// Builder が Modes から除外する規則と一致することを確認します。
// 判定が二重に書かれている以上、ずれれば利用側の一覧と Build の対象が食い違います。
func TestIsPartial_MatchesBuilder(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/review.md":     &fstest.MapFile{Data: []byte("レビュー")},
		"prompts/_output.md":    &fstest.MapFile{Data: []byte("出力形式")},
		"prompts/en/rock.md":    &fstest.MapFile{Data: []byte("rock")},
		"prompts/en/_output.md": &fstest.MapFile{Data: []byte("output")},
	}

	b, err := LoadFS(mockFS, "prompts", WithRecursive())
	require.NoError(t, err)

	for _, entry := range []string{"review", "_output", "en/rock", "en/_output"} {
		assert.Equal(t, !IsPartial(entry, DefaultPartialPrefix), b.Has(entry),
			"%s の partial 判定が Builder と一致しません", entry)
	}
}
