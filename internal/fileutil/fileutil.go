package fileutil

import (
	"path/filepath"
	"strings"
)

// NormalizeFilename normalizes a filename for FAT32 compatibility
// by replacing invalid characters with underscores
func NormalizeFilename(filename string) string {
	// Remove extension
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Replace invalid FAT32 chars
	invalid := []string{"\\", "/", ":", ";", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, char, "_")
	}

	return nameWithoutExt
}

// GetOutputFilename converts an input filename to the expected output filename
// with the given format extension
func GetOutputFilename(inputFilename, format string) string {
	normalized := NormalizeFilename(inputFilename)

	// Add extension based on format
	var ext string
	if format == "jpg" || format == "jpeg" {
		ext = ".jpg"
	} else {
		ext = ".webp"
	}

	return normalized + ext
}
