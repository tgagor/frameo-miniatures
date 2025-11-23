package pruner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tgagor/frameo-miniatures/internal/discovery"
)

func TestPruner_Prune(t *testing.T) {
	// Setup temp directories
	tmpDir, err := os.MkdirTemp("", "pruner-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err = os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create input files
	inputFiles := []string{
		"photo1.jpg",
		"photo2.jpg",
		"subdir/photo3.jpg",
	}

	for _, f := range inputFiles {
		path := filepath.Join(inputDir, f)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create output files (some orphaned)
	outputFiles := []string{
		"photo1.webp",           // Valid
		"photo2.webp",           // Valid
		"subdir/photo3.webp",    // Valid
		"orphaned.webp",         // Should be removed
		"subdir/orphaned2.webp", // Should be removed
	}

	for _, f := range outputFiles {
		path := filepath.Join(outputDir, f)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create pruner
	matcher := &discovery.IgnoreMatcher{}
	pruner := NewPruner(inputDir, outputDir, "webp", matcher, false)

	// Run pruning
	removedCount, err := pruner.Prune()
	require.NoError(t, err)

	// Should have removed 2 orphaned files
	assert.Equal(t, 2, removedCount)

	// Verify expected files still exist
	assert.FileExists(t, filepath.Join(outputDir, "photo1.webp"))
	assert.FileExists(t, filepath.Join(outputDir, "photo2.webp"))
	assert.FileExists(t, filepath.Join(outputDir, "subdir/photo3.webp"))

	// Verify orphaned files were removed
	assert.NoFileExists(t, filepath.Join(outputDir, "orphaned.webp"))
	assert.NoFileExists(t, filepath.Join(outputDir, "subdir/orphaned2.webp"))
}

func TestPruner_PruneWithIgnorePattern(t *testing.T) {
	// Setup temp directories
	tmpDir, err := os.MkdirTemp("", "pruner-ignore-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err = os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create .frameoignore
	ignoreContent := `ignored/*`
	ignoreFile := filepath.Join(inputDir, ".frameoignore")
	err = os.WriteFile(ignoreFile, []byte(ignoreContent), 0644)
	require.NoError(t, err)

	// Create input files
	inputFiles := []string{
		"photo1.jpg",
		"ignored/photo2.jpg", // This will be ignored
	}

	for _, f := range inputFiles {
		path := filepath.Join(inputDir, f)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create output files
	outputFiles := []string{
		"photo1.webp",         // Valid
		"ignored/photo2.webp", // Should be removed (ignored in input)
	}

	for _, f := range outputFiles {
		path := filepath.Join(outputDir, f)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create matcher with ignore patterns
	matcher, err := discovery.NewIgnoreMatcher(ignoreFile, inputDir)
	require.NoError(t, err)

	pruner := NewPruner(inputDir, outputDir, "webp", matcher, false)

	// Run pruning
	removedCount, err := pruner.Prune()
	require.NoError(t, err)

	// Should have removed 1 file (the ignored one)
	assert.Equal(t, 1, removedCount)

	// Verify expected file still exists
	assert.FileExists(t, filepath.Join(outputDir, "photo1.webp"))

	// Verify ignored file was removed
	assert.NoFileExists(t, filepath.Join(outputDir, "ignored/photo2.webp"))
}

func TestPruner_FilenameNormalization(t *testing.T) {
	// Setup temp directories
	tmpDir, err := os.MkdirTemp("", "pruner-normalize-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err = os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create input file with FAT32-invalid characters
	// Note: We can't actually create files with these chars on most filesystems,
	// so we simulate what the processor would do
	inputFiles := []string{
		"photo_test.jpg", // Simulates "photo:test.jpg" after normalization
	}

	for _, f := range inputFiles {
		path := filepath.Join(inputDir, f)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create output files - one normalized, one not
	// Only create the normalized one since we can't create files with : on most systems
	path := filepath.Join(outputDir, "photo_test.webp")
	err = os.WriteFile(path, []byte("test"), 0644)
	require.NoError(t, err)

	// Create pruner
	matcher := &discovery.IgnoreMatcher{}
	pruner := NewPruner(inputDir, outputDir, "webp", matcher, false)

	// Run pruning
	removedCount, err := pruner.Prune()
	require.NoError(t, err)

	// Should not have removed anything (normalized file matches)
	assert.Equal(t, 0, removedCount)

	// Verify normalized file still exists
	assert.FileExists(t, filepath.Join(outputDir, "photo_test.webp"))
}

func TestPruner_FormatConversion(t *testing.T) {
	// Test that pruner correctly handles format conversion (jpg -> webp)
	tmpDir, err := os.MkdirTemp("", "pruner-format-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err = os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create input JPG file
	inputPath := filepath.Join(inputDir, "photo.jpg")
	err = os.WriteFile(inputPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Create output WebP file (correct) and orphaned JPG (should be removed)
	outputWebP := filepath.Join(outputDir, "photo.webp")
	err = os.WriteFile(outputWebP, []byte("test"), 0644)
	require.NoError(t, err)

	outputJPG := filepath.Join(outputDir, "photo.jpg")
	err = os.WriteFile(outputJPG, []byte("test"), 0644)
	require.NoError(t, err)

	// Create pruner with webp format
	matcher := &discovery.IgnoreMatcher{}
	pruner := NewPruner(inputDir, outputDir, "webp", matcher, false)

	// Run pruning
	removedCount, err := pruner.Prune()
	require.NoError(t, err)

	// Should have removed the JPG (wrong format)
	assert.Equal(t, 1, removedCount)

	// Verify WebP still exists, JPG was removed
	assert.FileExists(t, outputWebP)
	assert.NoFileExists(t, outputJPG)
}
