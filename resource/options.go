package resource

import "strings"

// config は Load の動作設定を保持します。
type config struct {
	recursive  bool
	extensions []string
}

// Option は Load の動作を変更する関数型です。
type Option func(*config)

// WithRecursive は、サブディレクトリを再帰的に走査します。
// このときモード名は rootDir からの相対パスから拡張子を除いたものになります
// （例: rootDir が "prompts" で "prompts/en/rock.md" なら "en/rock"）。
func WithRecursive() Option {
	return func(c *config) {
		c.recursive = true
	}
}

// WithExtensions は、読み込む対象を指定した拡張子のファイルだけに限定します。
// 先頭のドットは省略可能です（".md" と "md" は同じ扱いです）。
// 指定しない場合はすべての拡張子が対象になります。
func WithExtensions(exts ...string) Option {
	return func(c *config) {
		for _, ext := range exts {
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			c.extensions = append(c.extensions, ext)
		}
	}
}

// matchExtension は、ファイル名が対象の拡張子に一致するかを判定します。
func (c *config) matchExtension(ext string) bool {
	if len(c.extensions) == 0 {
		return true
	}
	for _, want := range c.extensions {
		if strings.EqualFold(ext, want) {
			return true
		}
	}
	return false
}
