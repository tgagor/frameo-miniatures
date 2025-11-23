package processor

import (
	"os"
	"path/filepath"
	"testing"

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
	proc := NewProcessor(1000, 500, 80, "webp")

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
	proc := NewProcessor(1280, 800, 80, "webp")

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
