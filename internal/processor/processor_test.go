package processor

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"
)

func TestProcessor_ProcessFile(t *testing.T) {
	// Use the real example file
	exampleFile := "../../example/IMG_20220811_094859.jpg"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("Example file not found, skipping test")
	}

	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-proc-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Copy example file
	srcPath := filepath.Join(srcDir, "test.jpg")
	input, err := os.ReadFile(exampleFile)
	require.NoError(t, err)
	err = os.WriteFile(srcPath, input, 0644)
	require.NoError(t, err)

	// Initialize Processor
	// Example file is 6016x3384, aspect ratio ~1.78:1
	// Target 1000x500 has aspect ratio 2:1
	// Should fit to 889x500 to preserve aspect ratio
	proc := NewProcessor(1000, 500, 80, "webp", false)

	// Process
	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	// Check output
	destPath := filepath.Join(destDir, "test.webp")
	assert.FileExists(t, destPath)

	// Verify dimensions
	f, err := os.Open(destPath)
	require.NoError(t, err)
	defer f.Close()

	config, err := webp.DecodeConfig(f)
	require.NoError(t, err)

	// Should fit within 1000x500
	assert.LessOrEqual(t, config.Width, 1000)
	assert.LessOrEqual(t, config.Height, 500)

	// Verify aspect ratio is preserved (approximately)
	sourceAspect := 6016.0 / 3384.0
	outputAspect := float64(config.Width) / float64(config.Height)
	assert.InDelta(t, sourceAspect, outputAspect, 0.01, "Aspect ratio should be preserved")
}

func TestProcessor_ProcessFile_AspectPreservation(t *testing.T) {
	// Use the real example file
	exampleFile := "../../example/IMG_20220811_094859.jpg"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("Example file not found, skipping test")
	}

	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-proc-test-aspect")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Copy example file
	srcPath := filepath.Join(srcDir, "test.jpg")
	input, err := os.ReadFile(exampleFile)
	require.NoError(t, err)
	err = os.WriteFile(srcPath, input, 0644)
	require.NoError(t, err)

	// Target: 1280x800
	// Example file is 6016x3384 (aspect ~1.78:1)
	// Should fit to 1280x720 to preserve aspect ratio
	proc := NewProcessor(1280, 800, 80, "webp", false)

	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	destPath := filepath.Join(destDir, "test.webp")
	f, err := os.Open(destPath)
	require.NoError(t, err)
	defer f.Close()

	config, err := webp.DecodeConfig(f)
	require.NoError(t, err)

	// Should fit within 1280x800
	assert.LessOrEqual(t, config.Width, 1280)
	assert.LessOrEqual(t, config.Height, 800)

	// Verify aspect ratio is preserved
	sourceAspect := 6016.0 / 3384.0
	outputAspect := float64(config.Width) / float64(config.Height)
	assert.InDelta(t, sourceAspect, outputAspect, 0.01, "Aspect ratio should be preserved")
}

func TestProcessor_ProcessFile_PortraitOptimization(t *testing.T) {
	// Use the real example file
	exampleFile := "../../example/IMG_20220811_094859.jpg"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("Example file not found, skipping test")
	}

	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "frameo-proc-test-portrait")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	// Create a portrait image by rotating the example
	f, err := os.Open(exampleFile)
	require.NoError(t, err)
	img, _, err := image.Decode(f)
	f.Close()
	require.NoError(t, err)

	portraitImg := imaging.Rotate90(img) // 6016x3384 -> 3384x6016

	srcPath := filepath.Join(srcDir, "portrait.jpg")
	outF, err := os.Create(srcPath)
	require.NoError(t, err)
	err = jpeg.Encode(outF, portraitImg, nil)
	outF.Close()
	require.NoError(t, err)

	// Initialize Processor with Landscape Frame 1280x800
	// Old behavior: Fit 3384x6016 into 1280x800 -> Scale H: 800/6016=0.13 -> 450x800.
	// New behavior: Fit 3384x6016 into 800x1280 -> Scale H: 1280/6016=0.21 -> 720x1280.

	// So height should be 1280 (or close to it), which is > 800.

	proc := NewProcessor(1280, 800, 80, "webp", false)

	err = proc.ProcessFile(srcPath, destDir)
	require.NoError(t, err)

	destPath := filepath.Join(destDir, "portrait.webp")
	destF, err := os.Open(destPath)
	require.NoError(t, err)
	defer destF.Close()

	config, err := webp.DecodeConfig(destF)
	require.NoError(t, err)

	// Verify dimensions
	// Should be optimized for portrait mode (800x1280)
	// So Height should be > 800 (the landscape height)
	assert.Greater(t, config.Height, 800, "Portrait image should use the longer dimension of the frame")
	assert.LessOrEqual(t, config.Height, 1280)
	assert.LessOrEqual(t, config.Width, 800)
}
