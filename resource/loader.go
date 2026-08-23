// Package resource は、ファイルシステム上のディレクトリからファイルを読み込み、
// 名前をキーとするマップとして返すユーティリティを提供します。
//
// 対象の絞り込み（接頭辞・拡張子）とサブディレクトリの走査はオプションで指定します。
// 主な用途は embed.FS からのプロンプト読み込みですが、fs.FS であれば何でも扱えます。
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

// Load は指定されたファイルシステム内のディレクトリからファイルを読み込み、
// モード名（既定では拡張子を除いたファイル名）をキーとするマップとして返します。
// 既定ではサブディレクトリを走査しません。対象の絞り込みと走査方法は
// WithPrefix / WithExtensions / WithRecursive で変更します。
//
//	templates, err := resource.Load(promptFiles, "prompts")
//	templates, err := resource.Load(promptFiles, "prompts", resource.WithExtensions(".md"))
func Load(fileSystem fs.FS, rootDir string, opts ...Option) (map[string]string, error) {
	cfg := newConfig(opts...)

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
		if !strings.HasPrefix(fileName, cfg.prefix) {
			return nil
		}
		if !cfg.matchExtension(path.Ext(fileName)) {
			return nil
		}

		modeName := modeNameOf(rootDir, filePath, cfg.prefix)

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
