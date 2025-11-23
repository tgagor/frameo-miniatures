package discovery

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// File represents a file to be processed
type File struct {
	Path         string
	RelativePath string
}

// WalkFiles walks the input directory and sends valid files to the files channel.
// It closes the channel when done.
func WalkFiles(root string, files chan<- File, matcher *IgnoreMatcher) {
	defer close(files)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("Error walking path")
			return nil // Continue walking
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("Error getting relative path")
			return nil
		}

		// Check ignore rules
		if matcher.Matches(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isValidExtension(path) {
			files <- File{
				Path:         path,
				RelativePath: relPath,
			}
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error walking directory")
	}
}

func isValidExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".heic"
}
