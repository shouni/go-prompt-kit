// Package resource provides a utility that reads files from a directory on a file system and
// returns them as a map keyed by name.
//
// Narrowing the targets (prefix, extension) and walking subdirectories are set by options. The main
// use is loading prompts from an embed.FS, but any fs.FS works.
package resource

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ErrNotDirectory is returned when rootDir names something other than a directory.
var ErrNotDirectory = errors.New("resource: not a directory")

// Load reads files from a directory in the given file system and returns them as a map keyed by
// mode name (by default, the file name without its extension). By default it does not walk
// subdirectories. WithPrefix / WithExtensions / WithRecursive change the targets and how they are
// walked.
//
//	templates, err := resource.Load(promptFiles, "prompts")
//	templates, err := resource.Load(promptFiles, "prompts", resource.WithExtensions(".md"))
func Load(fileSystem fs.FS, rootDir string, opts ...Option) (map[string]string, error) {
	cfg := newConfig(opts...)

	// fs.WalkDir also accepts a file as its starting point, so check first that it is a directory.
	info, err := fs.Stat(fileSystem, rootDir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", rootDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("read directory %s: %w", rootDir, ErrNotDirectory)
	}

	templates := make(map[string]string)

	walkErr := fs.WalkDir(fileSystem, rootDir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("read directory %s: %w", filePath, err)
		}

		if entry.IsDir() {
			// When not recursive, do not enter subdirectories directly under rootDir.
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
			return fmt.Errorf("read file %s: %w", filePath, err)
		}

		if _, exists := templates[modeName]; exists {
			return fmt.Errorf("template name collision: %s (file: %s)", modeName, filePath)
		}
		templates[modeName] = string(content)

		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return templates, nil
}

// modeNameOf derives the mode name from a file path. A file directly under rootDir gets its file
// name without the prefix and extension; a file in a subdirectory keeps its path relative to
// rootDir (e.g. rootDir="prompts", filePath="prompts/en/rock.md" gives "en/rock").
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
