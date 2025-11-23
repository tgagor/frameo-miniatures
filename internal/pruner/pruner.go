package pruner

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/tgagor/frameo-miniatures/internal/discovery"
	"github.com/tgagor/frameo-miniatures/internal/fileutil"
)

// Pruner handles cleanup of output directory
type Pruner struct {
	InputDir  string
	OutputDir string
	Format    string
	Matcher   *discovery.IgnoreMatcher
	DryRun    bool
}

// NewPruner creates a new pruner
func NewPruner(inputDir, outputDir, format string, matcher *discovery.IgnoreMatcher, dryRun bool) *Pruner {
	return &Pruner{
		InputDir:  inputDir,
		OutputDir: outputDir,
		Format:    format,
		Matcher:   matcher,
		DryRun:    dryRun,
	}
}

// Prune removes files from output that don't exist in input or match ignore patterns
func (p *Pruner) Prune() (int, error) {
	// Build a set of expected output files based on input
	expectedFiles := make(map[string]bool)

	// Walk input directory to find all valid source files
	files := make(chan discovery.File, 1000)
	go discovery.WalkFiles(p.InputDir, files, p.Matcher)

	for file := range files {
		// Determine what the output filename would be
		outputRelPath := p.getOutputPath(file.RelativePath)
		expectedFiles[outputRelPath] = true
	}

	// Walk output directory and remove files not in expected set
	removedCount := 0
	err := filepath.Walk(p.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from output dir
		relPath, err := filepath.Rel(p.OutputDir, path)
		if err != nil {
			return err
		}

		// Check if this file should exist
		if !expectedFiles[relPath] {
			if p.DryRun {
				log.Info().Str("file", relPath).Msg("[DRY RUN] Would prune orphaned file")
				removedCount++
			} else {
				log.Info().Str("file", relPath).Msg("Pruning orphaned file")
				if err := os.Remove(path); err != nil {
					log.Warn().Err(err).Str("file", path).Msg("Failed to remove file")
				} else {
					removedCount++
				}
			}
		}

		return nil
	})

	// Clean up empty directories
	if err == nil {
		p.removeEmptyDirs(p.OutputDir)
	}

	return removedCount, err
}

// getOutputPath converts input relative path to expected output relative path
func (p *Pruner) getOutputPath(inputRelPath string) string {
	// Get the directory and filename
	dir := filepath.Dir(inputRelPath)
	filename := filepath.Base(inputRelPath)

	// Use shared utility to get normalized output filename
	outputFilename := fileutil.GetOutputFilename(filename, p.Format)
	return filepath.Join(dir, outputFilename)
}

// removeEmptyDirs recursively removes empty directories
func (p *Pruner) removeEmptyDirs(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == dir {
			return err
		}

		if p.DryRun {
			// In dry-run mode, just check if directory is empty
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) == 0 {
				log.Debug().Str("dir", path).Msg("[DRY RUN] Would remove empty directory")
			}
		} else {
			// Try to remove directory (will only succeed if empty)
			if err := os.Remove(path); err == nil {
				log.Debug().Str("dir", path).Msg("Removed empty directory")
			}
		}

		return nil
	})
}
