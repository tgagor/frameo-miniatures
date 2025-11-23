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
	"golang.org/x/image/webp"
)

func TestProcessor_ProcessFile(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-proc-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a dummy large image (2000x1000)
	img := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	// Fill with some color
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	srcPath := filepath.Join(srcDir, "test.jpg")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	err = jpeg.Encode(f, img, nil)
	f.Close()
	require.NoError(t, err)

	// Initialize Processor with target 1000x500
	// Aspect ratio of source is 2:1.
	// Target 1000x500 has aspect ratio 2:1.
	// Should fit exactly.
	proc := NewProcessor(1000, 500, 80)

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output
	destPath := filepath.Join(destDir, "test.webp")
	assert.FileExists(t, destPath)

	// Verify dimensions
	f, err = os.Open(destPath)
	require.NoError(t, err)
	defer f.Close()

	config, err := webp.DecodeConfig(f)
	require.NoError(t, err)

	assert.Equal(t, 1000, config.Width)
	assert.Equal(t, 500, config.Height)
}

func TestProcessor_ProcessFile_AspectPreservation(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-proc-test-aspect")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a dummy image (1000x1000) - Square
	img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	srcPath := filepath.Join(srcDir, "square.jpg")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	jpeg.Encode(f, img, nil)
	f.Close()

	// Target: 1280x800
	// Should be resized to fit within 1280x800.
	// Max height is 800. So it should be 800x800.
	proc := NewProcessor(1280, 800, 80)

	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	destPath := filepath.Join(destDir, "square.webp")
	f, err = os.Open(destPath)
	require.NoError(t, err)
	defer f.Close()

	config, err := webp.DecodeConfig(f)
	require.NoError(t, err)

	assert.Equal(t, 800, config.Width)
	assert.Equal(t, 800, config.Height)
}
