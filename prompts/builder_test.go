package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateBuilder_Build(t *testing.T) {
	testTemplates := map[string]string{
		"test_mode": "Review this diff:\n{{.DiffContent}}",
	}
	builder, err := NewBuilder(testTemplates)
	require.NoError(t, err)

	t.Run("Success_NilDataForStaticTemplate", func(t *testing.T) {
		staticTemplates := map[string]string{"static_mode": "Static Prompt"}
		staticBuilder, _ := NewBuilder(staticTemplates)

		result, err := staticBuilder.Build("static_mode", nil)
		assert.NoError(t, err)
		assert.Equal(t, "Static Prompt", result)
	})

	t.Run("Success_WithLocalStruct", func(t *testing.T) {
		type mockData struct {
			DiffContent string
		}
		data := mockData{
			DiffContent: "index 123..456\n+ func Hello() { fmt.Println(\"Hi\") }",
		}

		result, err := builder.Build("test_mode", data)
		assert.NoError(t, err)
		assert.Contains(t, result, "Review this diff:")
		assert.Contains(t, result, "func Hello()")
	})

	t.Run("Success_WithMap", func(t *testing.T) {
		data := map[string]string{
			"DiffContent": "map diff content",
		}
		result, err := builder.Build("test_mode", data)
		assert.NoError(t, err)
		assert.Contains(t, result, "map diff content")
	})

	t.Run("Error_UnknownMode", func(t *testing.T) {
		_, err := builder.Build("non_existent_mode", struct{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不明なモードです")
	})

	t.Run("Error_TemplateExecutionFailure", func(t *testing.T) {
		// テンプレートが期待するフィールド（DiffContent）を持たない構造体を渡す
		// text/template は存在しないフィールド参照でエラーを返すため、それを検証する
		data := struct{ WrongField string }{"oops"}
		result, err := builder.Build("test_mode", data)

		assert.Error(t, err)
		assert.Equal(t, "", result) // エラー時は空文字が返る実装
		assert.Contains(t, err.Error(), "プロンプトテンプレートの実行に失敗しました")
		assert.Contains(t, err.Error(), "can't evaluate field DiffContent")
	})
}

func TestNewBuilder_Validation(t *testing.T) {
	t.Run("Success_WithCustomTemplates", func(t *testing.T) {
		testTemplates := map[string]string{"mode1": "template content"}
		builder, err := NewBuilder(testTemplates)
		assert.NoError(t, err)
		assert.NotNil(t, builder)
	})

	t.Run("Error_NilOrEmptyTemplates", func(t *testing.T) {
		_, errNil := NewBuilder(nil)
		assert.Error(t, errNil)
		assert.Contains(t, errNil.Error(), "テンプレートマップが空またはnilです")

		_, errEmpty := NewBuilder(map[string]string{})
		assert.Error(t, errEmpty)
		assert.Contains(t, errEmpty.Error(), "テンプレートマップが空またはnilです")
	})
}
