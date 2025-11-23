package discovery

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	gitignore "github.com/sabhiram/go-gitignore"
)

// IgnoreMatcher checks if a file should be ignored
type IgnoreMatcher struct {
	ignorer *gitignore.GitIgnore
}

// NewIgnoreMatcher creates a new matcher from a .frameoignore file
func NewIgnoreMatcher(root string) (*IgnoreMatcher, error) {
	ignorePath := filepath.Join(root, ".frameoignore")
	log.Info().Str("path", ignorePath).Msg("Loading .frameoignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		return &IgnoreMatcher{ignorer: nil}, nil
	}

	ignorer, err := gitignore.CompileIgnoreFile(ignorePath)
	if err != nil {
		return nil, err
	}

	return &IgnoreMatcher{ignorer: ignorer}, nil
}

// Matches returns true if the path should be ignored
func (m *IgnoreMatcher) Matches(path string, isDir bool) bool {
	if m.ignorer == nil {
		return false
	}
	return m.ignorer.MatchesPath(path)
}
