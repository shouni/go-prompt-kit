// Package resource は、埋め込みファイルシステムからプレフィックス一致するファイルを
// 読み込むユーティリティを提供します。
package resource

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ErrNotDirectory は、rootDir にディレクトリ以外が指定された場合に返されます。
var ErrNotDirectory = errors.New("ディレクトリではありません")

// Load は指定されたファイルシステム内のディレクトリから、指定された接頭辞を持つファイルを読み込み、マップとして返します。
// 既定ではサブディレクトリを走査しません。WithRecursive を指定すると再帰的に読み込みます。
func Load(fileSystem fs.FS, rootDir, prefix string, opts ...Option) (map[string]string, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	// fs.WalkDir はファイルを起点にしても走査できてしまうため、
	// ディレクトリであることを先に確認して従来どおりエラーにします。
	info, err := fs.Stat(fileSystem, rootDir)
	if err != nil {
		return nil, fmt.Errorf("ディレクトリ %s の読み込みに失敗: %w", rootDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("ディレクトリ %s の読み込みに失敗: %w", rootDir, ErrNotDirectory)
	}

	templates := make(map[string]string)

	walkErr := fs.WalkDir(fileSystem, rootDir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("ディレクトリ %s の読み込みに失敗: %w", filePath, err)
		}

		if entry.IsDir() {
			// 非再帰の場合、rootDir 直下のサブディレクトリには入りません。
			if filePath != rootDir && !cfg.recursive {
				return fs.SkipDir
			}
			return nil
		}

		fileName := entry.Name()
		if !strings.HasPrefix(fileName, prefix) {
			return nil
		}
		if !cfg.matchExtension(path.Ext(fileName)) {
			return nil
		}

		modeName := modeNameOf(rootDir, filePath, prefix)

		content, err := fs.ReadFile(fileSystem, filePath)
		if err != nil {
			return fmt.Errorf("ファイル %s の読み込みに失敗: %w", filePath, err)
		}

		if _, exists := templates[modeName]; exists {
			return fmt.Errorf("テンプレート名が衝突しています: %s (ファイル: %s)", modeName, filePath)
		}
		templates[modeName] = string(content)

		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return templates, nil
}

// modeNameOf は、ファイルパスからモード名を導出します。
// rootDir 直下のファイルはファイル名から接頭辞と拡張子を除いたもの、
// サブディレクトリ内のファイルは rootDir からの相対パスを保った名前になります
// （例: rootDir="prompts", filePath="prompts/en/rock.md" なら "en/rock"）。
func modeNameOf(rootDir, filePath, prefix string) string {
	relPath := filePath
	if rootDir != "." {
		relPath = strings.TrimPrefix(filePath, rootDir+"/")
	}

	baseName := path.Base(relPath)
	modeName := strings.TrimSuffix(
		strings.TrimPrefix(baseName, prefix),
		path.Ext(baseName),
	)

	if dir := path.Dir(relPath); dir != "." {
		modeName = dir + "/" + modeName
	}

	return modeName
}
