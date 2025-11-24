package processor

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessor_ProcessFile_JPEG_Output(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-jpeg-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 1000, 800))
	// Fill with some color
	for y := 0; y < 800; y++ {
		for x := 0; x < 1000; x++ {
			img.Set(x, y, color.RGBA{255, 100, 50, 255})
		}
	}

	srcPath := filepath.Join(srcDir, "test.jpg")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	err = jpeg.Encode(f, img, nil)
	f.Close()
	require.NoError(t, err)

	// Initialize Processor with JPEG format
	proc := NewProcessor(800, 600, 85, "jpg", false)

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output exists with .jpg extension
	destPath := filepath.Join(destDir, "test.jpg")
	assert.FileExists(t, destPath)

	// Verify it's a valid JPEG
	f, err = os.Open(destPath)
	require.NoError(t, err)
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	require.NoError(t, err)

	// Verify dimensions (should fit within 800x600)
	assert.LessOrEqual(t, config.Width, 800)
	assert.LessOrEqual(t, config.Height, 600)
}

func TestProcessor_ProcessFile_WebP_Output(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-webp-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 1000, 800))
	srcPath := filepath.Join(srcDir, "test.jpg")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	jpeg.Encode(f, img, nil)
	f.Close()

	// Initialize Processor with WebP format
	proc := NewProcessor(800, 600, 80, "webp", false)

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output exists with .webp extension
	destPath := filepath.Join(destDir, "test.webp")
	assert.FileExists(t, destPath)
}
