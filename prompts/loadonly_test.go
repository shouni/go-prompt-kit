package prompts

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBuilder_RejectsLoadOnlyOptions confirms that NewBuilder does not accept options that are
// only valid while loading.
//
// Ignored silently, every entry would be registered while the caller believes it filtered them, and
// it would still compile and run.
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
			assert.Contains(t, err.Error(), tt.wantName, "should name the offending option")
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

	// LoadFS consumes the load-only options itself, so they do not trip the check.
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

// TestIsPartial_MatchesBuilder confirms that the exported IsPartial agrees with the rule Builder
// uses to exclude entries from Modes. As long as the predicate is written twice, a drift would make
// a consumer's list disagree with what Build accepts.
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
			"partial detection for %s disagrees with Builder", entry)
	}
}
