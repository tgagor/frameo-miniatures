package processor

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dsoprea/go-exif/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessor_ProcessFile_PreservesEXIF(t *testing.T) {
	// Use the real example file if it exists
	exampleFile := "../../example/IMG_20220811_094859.jpg"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("Example file not found, skipping EXIF test")
	}

	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-exif-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Copy example file to src
	srcPath := filepath.Join(srcDir, "test_with_exif.jpg")
	input, err := os.ReadFile(exampleFile)
	require.NoError(t, err)
	err = os.WriteFile(srcPath, input, 0644)
	require.NoError(t, err)

	// Initialize Processor
	proc := NewProcessor(800, 600, 80, "webp")

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output exists
	destPath := filepath.Join(destDir, "test_with_exif.webp")
	assert.FileExists(t, destPath)

	// Verify EXIF data is preserved
	destFile, err := os.Open(destPath)
	require.NoError(t, err)
	defer func() { _ = destFile.Close() }()

	// Read WebP and check for EXIF
	rawExif, err := exif.SearchAndExtractExifWithReader(destFile)
	require.NoError(t, err, "EXIF data should be present in output WebP")

	// Parse EXIF and check DateTimeOriginal
	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	require.NoError(t, err)

	foundDate := false
	for _, tag := range entries {
		if tag.TagName == "DateTimeOriginal" {
			assert.Equal(t, "2022:08:11 09:49:00", tag.FormattedFirst)
			foundDate = true
			break
		}
	}
	assert.True(t, foundDate, "DateTimeOriginal should be preserved")

	// Verify file modification time
	info, err := os.Stat(destPath)
	require.NoError(t, err)

	expectedTime, _ := time.Parse("2006:01:02 15:04:05", "2022:08:11 09:49:00")
	// Allow some tolerance for time comparison (1 second)
	timeDiff := info.ModTime().Sub(expectedTime)
	assert.Less(t, timeDiff.Abs().Seconds(), 2.0, "File modification time should match EXIF date")
}

func TestProcessor_ProcessFile_NoEXIF(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-no-exif-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a test image WITHOUT EXIF data
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	srcPath := filepath.Join(srcDir, "test_no_exif.jpg")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	_ = jpeg.Encode(f, img, nil)
	_ = f.Close()

	// Get source file mod time
	srcInfo, err := os.Stat(srcPath)
	require.NoError(t, err)
	srcModTime := srcInfo.ModTime()

	// Initialize Processor
	proc := NewProcessor(400, 300, 80, "webp")

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output exists
	destPath := filepath.Join(destDir, "test_no_exif.webp")
	assert.FileExists(t, destPath)

	// Verify file modification time falls back to source mod time
	destInfo, err := os.Stat(destPath)
	require.NoError(t, err)

	timeDiff := destInfo.ModTime().Sub(srcModTime)
	assert.Less(t, timeDiff.Abs().Seconds(), 2.0, "File modification time should match source when no EXIF")
}

func TestProcessor_ProcessFile_JPEG_PreservesEXIF(t *testing.T) {
	// Use the real example file if it exists
	exampleFile := "../../example/IMG_20220811_094859.jpg"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("Example file not found, skipping JPEG EXIF test")
	}

	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-jpeg-exif-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Copy example file to src
	srcPath := filepath.Join(srcDir, "test_with_exif.jpg")
	input, err := os.ReadFile(exampleFile)
	require.NoError(t, err)
	err = os.WriteFile(srcPath, input, 0644)
	require.NoError(t, err)

	// Initialize Processor with JPEG format
	proc := NewProcessor(800, 600, 80, "jpg")

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output exists
	destPath := filepath.Join(destDir, "test_with_exif.jpg")
	assert.FileExists(t, destPath)

	// Verify EXIF data is preserved in JPEG
	destFile, err := os.Open(destPath)
	require.NoError(t, err)
	defer func() { _ = destFile.Close() }()

	// Read JPEG and check for EXIF
	rawExif, err := exif.SearchAndExtractExifWithReader(destFile)
	require.NoError(t, err, "EXIF data should be present in output JPEG")

	// Parse EXIF and check DateTimeOriginal
	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	require.NoError(t, err)

	foundDate := false
	for _, tag := range entries {
		if tag.TagName == "DateTimeOriginal" {
			assert.Equal(t, "2022:08:11 09:49:00", tag.FormattedFirst)
			foundDate = true
			break
		}
	}
	assert.True(t, foundDate, "DateTimeOriginal should be preserved in JPEG")
}
