package frontmatter_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/frontmatter"
)

// modeInfo stands in for the metadata struct a consumer defines itself. This package does not care
// about the metadata format, so the tests pass the standard library's json.Unmarshal as the
// UnmarshalFunc.
type modeInfo struct {
	Direction string   `json:"direction"`
	UseWhen   string   `json:"use_when"`
	Sections  []string `json:"sections"`
}

func TestDecode(t *testing.T) {
	t.Run("decodes front matter into a struct", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(`{"direction":"技術解説","use_when":"仕様を説明するとき"}`, &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, modeInfo{Direction: "技術解説", UseWhen: "仕様を説明するとき"}, got)
	})

	t.Run("empty front matter succeeds without decoding", func(t *testing.T) {
		got := modeInfo{Direction: "既存の値"}
		err := frontmatter.Decode("", &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, "既存の値", got.Direction, "the value was overwritten")
	})

	t.Run("whitespace-only front matter counts as empty", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(" \n\t\n", &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("returns an error when parsing fails", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(`{"direction":`, &got, json.Unmarshal)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse front matter")
	})

	t.Run("wraps the parse function's error", func(t *testing.T) {
		sentinel := errors.New("parser-specific failure")
		fail := func([]byte, any) error { return sentinel }

		err := frontmatter.Decode("direction: 技術解説", &modeInfo{}, fail)

		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("returns an error without a parse function", func(t *testing.T) {
		err := frontmatter.Decode(`{"direction":"技術解説"}`, &modeInfo{}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})

	// Empty data would hide a wiring mistake, so the check runs whatever the front matter holds.
	t.Run("detects a missing parse function even with empty front matter", func(t *testing.T) {
		err := frontmatter.Decode("", &modeInfo{}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})
}

func TestDecodeAs(t *testing.T) {
	t.Run("returns front matter as T", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":"技術解説"}`, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, modeInfo{Direction: "技術解説"}, got)
	})

	t.Run("empty front matter returns the zero value", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo]("", json.Unmarshal)

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("returns the zero value and an error when parsing fails", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":`, json.Unmarshal)

		require.Error(t, err)
		assert.Zero(t, got, "a partially read value was returned on failure")
	})

	t.Run("returns an error without a parse function", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":"技術解説"}`, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
		assert.Zero(t, got)
	})

	t.Run("matches Decode", func(t *testing.T) {
		const front = `{"direction":"技術解説","sections":["a","b"]}`

		var want modeInfo
		require.NoError(t, frontmatter.Decode(front, &want, json.Unmarshal))

		got, err := frontmatter.DecodeAs[modeInfo](front, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestDecodeMap(t *testing.T) {
	t.Run("decodes per key", func(t *testing.T) {
		fronts := map[string]string{
			"tech_solo": `{"direction":"技術解説","sections":["Intro","Body"]}`,
			"news":      `{"direction":"時事"}`,
		}

		got, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, map[string]modeInfo{
			"tech_solo": {Direction: "技術解説", Sections: []string{"Intro", "Body"}},
			"news":      {Direction: "時事"},
		}, got)
	})

	t.Run("entries without front matter stay zero", func(t *testing.T) {
		fronts := map[string]string{
			"tech_solo": `{"direction":"技術解説"}`,
			"plain":     "",
		}

		got, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

		require.NoError(t, err)
		require.Contains(t, got, "plain", "the keys should match the input")
		assert.Zero(t, got["plain"])
	})

	t.Run("error names the failed keys", func(t *testing.T) {
		fronts := map[string]string{
			"tech_solo": `{"direction":"技術解説"}`,
			"broken":    `{"direction":`,
		}

		_, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "'broken'")
	})

	// Confirms that with several broken entries the reported key does not change between runs. Left
	// to map iteration order, a fixed error would appear to recur on another key.
	t.Run("reported keys are stable in name order", func(t *testing.T) {
		fronts := map[string]string{
			"aaa": `{"direction":`,
			"mmm": `{"direction":`,
			"zzz": `{"direction":`,
		}

		for range 20 {
			_, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "'aaa'")
		}
	})

	t.Run("returns an error without a parse function", func(t *testing.T) {
		_, err := frontmatter.DecodeMap[modeInfo](map[string]string{"a": ""}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})

	t.Run("empty input returns an empty map", func(t *testing.T) {
		got, err := frontmatter.DecodeMap[modeInfo](nil, json.Unmarshal)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

// TestSplitMapAndDecodeMap confirms that splitting body and metadata, starting from what
// resource.Load returns, works end to end.
func TestSplitMapAndDecodeMap(t *testing.T) {
	files := map[string]string{
		"tech_solo": "---\n{\"direction\":\"技術解説\"}\n---\n本文A\n",
		"news":      "本文B\n",
	}

	bodies, fronts := frontmatter.SplitMap(files)
	infos, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)
	require.NoError(t, err)

	assert.Equal(t, "本文A\n", bodies["tech_solo"])
	assert.Equal(t, "本文B\n", bodies["news"])
	assert.Equal(t, "技術解説", infos["tech_solo"].Direction)
	assert.Zero(t, infos["news"])
}
