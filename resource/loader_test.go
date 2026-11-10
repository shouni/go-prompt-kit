package resource

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	mockFS := fstest.MapFS{
		"prompts/prompt_review.md":   {Data: []byte("Review Template")},
		"prompts/prompt_release.md":  {Data: []byte("Release Template")},
		"prompts/other_file.txt":     {Data: []byte("Ignore Me")},
		"other_dir/prompt_unused.md": {Data: []byte("Wrong Dir")},
	}

	t.Run("success: loads prompts", func(t *testing.T) {
		templates, err := Load(mockFS, "prompts", WithPrefix("prompt_"))

		// A failing Load is fatal, so use require
		require.NoError(t, err, "Load failed")

		// Consistent assertions
		assert.Len(t, templates, 2, "expected 2 templates")
		assert.Equal(t, "Review Template", templates["review"], "unexpected review content")
		assert.Equal(t, "Release Template", templates["release"], "unexpected release content")
	})

	t.Run("error: missing directory", func(t *testing.T) {
		_, err := Load(mockFS, "non_existent", WithPrefix("prompt_"))

		// Verify that an error occurs
		assert.Error(t, err, "a missing directory should return an error")
	})
}
