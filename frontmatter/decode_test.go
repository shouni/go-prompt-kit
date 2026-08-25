package frontmatter_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shouni/go-prompt-kit/frontmatter"
)

// modeInfo は、利用側が自前で定義するメタデータ構造の代わりです。
// このパッケージはメタデータの書式に関知しないため、テストでは標準ライブラリの
// json.Unmarshal を UnmarshalFunc として渡します。
type modeInfo struct {
	Direction string   `json:"direction"`
	UseWhen   string   `json:"use_when"`
	Sections  []string `json:"sections"`
}

func TestDecode(t *testing.T) {
	t.Run("front matter を構造体へ読み取る", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(`{"direction":"技術解説","use_when":"仕様を説明するとき"}`, &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, modeInfo{Direction: "技術解説", UseWhen: "仕様を説明するとき"}, got)
	})

	t.Run("空の front matter は何もせず成功する", func(t *testing.T) {
		got := modeInfo{Direction: "既存の値"}
		err := frontmatter.Decode("", &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, "既存の値", got.Direction, "値が書き換えられています")
	})

	t.Run("空白だけの front matter も空として扱う", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(" \n\t\n", &got, json.Unmarshal)

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("解析に失敗した場合はエラーを返す", func(t *testing.T) {
		var got modeInfo
		err := frontmatter.Decode(`{"direction":`, &got, json.Unmarshal)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "front matter の解析に失敗しました")
	})

	t.Run("解析関数のエラーを包んで返す", func(t *testing.T) {
		sentinel := errors.New("解析器の都合")
		fail := func([]byte, any) error { return sentinel }

		err := frontmatter.Decode("direction: 技術解説", &modeInfo{}, fail)

		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("解析関数が無い場合はエラーを返す", func(t *testing.T) {
		err := frontmatter.Decode(`{"direction":"技術解説"}`, &modeInfo{}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})

	// 空のデータでは配線の誤りが表面化しないため、front matter の中身によらず検査します。
	t.Run("front matter が空でも解析関数の指定漏れは検出する", func(t *testing.T) {
		err := frontmatter.Decode("", &modeInfo{}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})
}

func TestDecodeAs(t *testing.T) {
	t.Run("front matter を T として返す", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":"技術解説"}`, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, modeInfo{Direction: "技術解説"}, got)
	})

	t.Run("空の front matter はゼロ値を返す", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo]("", json.Unmarshal)

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("解析に失敗した場合はゼロ値とエラーを返す", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":`, json.Unmarshal)

		require.Error(t, err)
		assert.Zero(t, got, "失敗した場合に途中まで読んだ値を返しています")
	})

	t.Run("解析関数が無い場合はエラーを返す", func(t *testing.T) {
		got, err := frontmatter.DecodeAs[modeInfo](`{"direction":"技術解説"}`, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
		assert.Zero(t, got)
	})

	t.Run("Decode と同じ結果になる", func(t *testing.T) {
		const front = `{"direction":"技術解説","sections":["a","b"]}`

		var want modeInfo
		require.NoError(t, frontmatter.Decode(front, &want, json.Unmarshal))

		got, err := frontmatter.DecodeAs[modeInfo](front, json.Unmarshal)

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestDecodeMap(t *testing.T) {
	t.Run("キーごとに読み取る", func(t *testing.T) {
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

	t.Run("front matter を持たないエントリはゼロ値で残る", func(t *testing.T) {
		fronts := map[string]string{
			"tech_solo": `{"direction":"技術解説"}`,
			"plain":     "",
		}

		got, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

		require.NoError(t, err)
		require.Contains(t, got, "plain", "キーの集合は入力と同じであるべきです")
		assert.Zero(t, got["plain"])
	})

	t.Run("失敗したキー名をエラーに含める", func(t *testing.T) {
		fronts := map[string]string{
			"tech_solo": `{"direction":"技術解説"}`,
			"broken":    `{"direction":`,
		}

		_, err := frontmatter.DecodeMap[modeInfo](fronts, json.Unmarshal)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "'broken'")
	})

	// 壊れたエントリが複数あっても、報告されるキーが実行のたびに変わらないことを確認します。
	// マップの走査順に任せると、直したはずのエラーが別のキーで再発したように見えます。
	t.Run("報告するキーは名前順で安定している", func(t *testing.T) {
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

	t.Run("解析関数が無い場合はエラーを返す", func(t *testing.T) {
		_, err := frontmatter.DecodeMap[modeInfo](map[string]string{"a": ""}, nil)

		assert.ErrorIs(t, err, frontmatter.ErrNoUnmarshalFunc)
	})

	t.Run("空の入力でも空のマップを返す", func(t *testing.T) {
		got, err := frontmatter.DecodeMap[modeInfo](nil, json.Unmarshal)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

// TestSplitMapAndDecodeMap は、resource.Load の戻り値を想定した
// 本文とメタデータの分離が一続きで動くことを確認します。
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
