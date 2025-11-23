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
// It searches in the following order:
// 1. Explicit path (if provided)
// 2. ~/.config/frameoignore
// 3. Input directory
// 4. Current directory
func NewIgnoreMatcher(explicitPath, inputDir string) (*IgnoreMatcher, error) {
	var ignorePath string

	// Helper to check if file exists
	exists := func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}

	if explicitPath != "" && exists(explicitPath) {
		ignorePath = explicitPath
	} else {
		// Check ~/.config/frameoignore
		home, err := os.UserHomeDir()
		if err == nil {
			configPath := filepath.Join(home, ".config", "frameoignore")
			if exists(configPath) {
				ignorePath = configPath
			}
		}

		// Check input dir
		if ignorePath == "" {
			inputIgnore := filepath.Join(inputDir, ".frameoignore")
			if exists(inputIgnore) {
				ignorePath = inputIgnore
			}
		}

		// Check current dir
		if ignorePath == "" {
			currentIgnore := ".frameoignore"
			if exists(currentIgnore) {
				ignorePath = currentIgnore
			}
		}
	}

	if ignorePath == "" {
		return &IgnoreMatcher{ignorer: nil}, nil
	}

	log.Info().Str("path", ignorePath).Msg("Loading .frameoignore")
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
