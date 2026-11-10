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

	t.Run("reports every one of several", func(t *testing.T) {
		_, err := NewBuilder(templates, WithRecursive(), WithExtensions(".md"))

		require.ErrorIs(t, err, ErrLoadOnlyOption)
		assert.Contains(t, err.Error(), "WithRecursive")
		assert.Contains(t, err.Error(), "WithExtensions")
	})

	t.Run("builder options still apply", func(t *testing.T) {
		_, err := NewBuilder(templates,
			WithDefaultMode("greet"),
			WithPartialPrefix("#"),
		)

		require.NoError(t, err)
	})

	// LoadFS consumes the load-only options itself, so they do not trip the check.
	t.Run("LoadFS accepts it", func(t *testing.T) {
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
		{name: "default prefix", entry: "_output", prefix: DefaultPartialPrefix, want: true},
		{name: "mode name", entry: "review", prefix: DefaultPartialPrefix, want: false},
		{name: "recursive path form", entry: "en/_output", prefix: DefaultPartialPrefix, want: true},
		{name: "mode name in path form", entry: "en/review", prefix: DefaultPartialPrefix, want: false},
		{name: "only the directory matches the prefix", entry: "_shared/review", prefix: DefaultPartialPrefix, want: false},
		{name: "custom prefix", entry: "compose_structure", prefix: "compose_", want: true},
		{name: "empty prefix matches nothing", entry: "_output", prefix: "", want: false},
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
