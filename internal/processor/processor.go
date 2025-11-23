package processor

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrium/goheif"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-exif/v3"
	"github.com/rs/zerolog/log"
)

// Processor handles image processing
type Processor struct {
	Width   int
	Height  int
	Quality int
	Format  string // "webp" or "jpg"
}

// NewProcessor creates a new processor
func NewProcessor(width, height, quality int, format string) *Processor {
	return &Processor{
		Width:   width,
		Height:  height,
		Quality: quality,
		Format:  format,
	}
}

// ProcessFile processes a single file
func (p *Processor) ProcessFile(srcPath, destDir string) error {
	// 1. Open file
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// 2. Decode image
	img, _, err := p.decode(f, srcPath)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// 3. Handle EXIF (Rotation & Date)
	var captureTime time.Time

	// Reset file pointer for EXIF search
	f.Seek(0, 0)
	rawExif, err := exif.SearchAndExtractExifWithReader(f)
	if err == nil {
		// Parse EXIF
		entries, _, err := exif.GetFlatExifData(rawExif, nil)
		if err == nil {
			for _, tag := range entries {
				if tag.TagName == "DateTimeOriginal" || tag.TagName == "CreateDate" {
					// Format: "2006:01:02 15:04:05"
					t, err := time.Parse("2006:01:02 15:04:05", tag.FormattedFirst)
					if err == nil {
						captureTime = t
						break
					}
				}
			}
		}
	}

	// Re-open file for imaging library (it needs path or reader, but let's use the decoded image if possible,
	// but imaging.Resize takes image.Image, so we are good).
	// However, we need to apply orientation.
	// If we decoded with standard lib, orientation is NOT applied.
	// We can use `imaging.FixOrientation` but it requires loading the image via `imaging.Open` or similar which handles EXIF.
	// Since we might have HEIC, we decoded manually.
	// Let's try to read orientation from EXIF and apply it.

	// Actually, `disintegration/imaging` has `FixOrientation` but it takes an image and we need to know the orientation.
	// Wait, `imaging.Open` supports many formats but maybe not HEIC by default?
	// Let's stick to manual decoding and then use `imaging` for resizing.

	// Auto-rotate
	img = p.fixOrientation(img, srcPath)

	// 4. Resize
	// "Fit Within" - imaging.Fit keeps aspect ratio
	img = imaging.Fit(img, p.Width, p.Height, imaging.CatmullRom)

	// 5. Normalize Filename
	destFilename := p.normalizeFilename(filepath.Base(srcPath))
	destPath := filepath.Join(destDir, destFilename)

	// Ensure dest dir exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create dest dir: %w", err)
	}

	// 6. Encode
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Encode based on format
	if p.Format == "jpg" || p.Format == "jpeg" {
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: p.Quality})
		out.Close()
		if err != nil {
			return fmt.Errorf("failed to encode jpeg: %w", err)
		}
	} else {
		// Default to WebP
		err = webp.Encode(out, img, &webp.Options{Quality: float32(p.Quality)})
		out.Close()
		if err != nil {
			return fmt.Errorf("failed to encode webp: %w", err)
		}
	}

	// 7. Preserve Metadata (EXIF and Time)
	// Copy EXIF data from source to destination (only for WebP, JPEG preserves it differently)
	if p.Format == "webp" {
		if err := p.copyExif(srcPath, destPath); err != nil {
			log.Warn().Err(err).Str("src", srcPath).Str("dest", destPath).Msg("Failed to copy EXIF data")
		}
	}
	// Note: For JPEG, we'd need a different approach to preserve EXIF
	// The standard library's jpeg.Encode doesn't preserve EXIF, so we'd need additional work

	// Also set file modification time
	if !captureTime.IsZero() {
		if err := os.Chtimes(destPath, time.Now(), captureTime); err != nil {
			log.Warn().Err(err).Str("path", destPath).Msg("Failed to set file time")
		}
	} else {
		// Fallback to source file mod time
		info, err := os.Stat(srcPath)
		if err == nil {
			os.Chtimes(destPath, time.Now(), info.ModTime())
		}
	}

	return nil
}

func (p *Processor) decode(r io.Reader, path string) (image.Image, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".heic" {
		img, err := goheif.Decode(r)
		return img, "heic", err
	}
	return image.Decode(r)
}

func (p *Processor) fixOrientation(img image.Image, path string) image.Image {
	// Read EXIF orientation
	f, err := os.Open(path)
	if err != nil {
		return img
	}
	defer f.Close()

	rawExif, err := exif.SearchAndExtractExifWithReader(f)
	if err != nil {
		return img
	}

	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {
		return img
	}

	var orientation int
	for _, tag := range entries {
		if tag.TagName == "Orientation" {
			if val, ok := tag.Value.([]uint16); ok && len(val) > 0 {
				orientation = int(val[0])
			} else if val, ok := tag.Value.([]uint8); ok && len(val) > 0 { // Sometimes it's byte
				orientation = int(val[0])
			}
			break
		}
	}

	// Apply rotation based on orientation
	// 1: Normal
	// 3: 180 rotate
	// 6: 90 CW
	// 8: 90 CCW
	switch orientation {
	case 3:
		return imaging.Rotate180(img)
	case 6:
		return imaging.Rotate270(img) // 90 CW is 270 CCW? No, Rotate270 is counter-clockwise?
		// imaging.Rotate270 rotates image 270 degrees counter-clockwise.
		// Orientation 6 is "The 0th row is at the visual right-hand side, and the 0th column is at the visual top." -> 90 CW.
		// 90 CW = 270 CCW. So yes.
	case 8:
		return imaging.Rotate90(img) // 90 CCW
	}
	return img
}

func (p *Processor) normalizeFilename(name string) string {
	// Remove extension
	ext := filepath.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, ext)

	// Replace invalid chars
	invalid := []string{"\\", "/", ":", ";", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, char, "_")
	}

	// Add extension based on format
	if p.Format == "jpg" || p.Format == "jpeg" {
		return nameWithoutExt + ".jpg"
	}
	return nameWithoutExt + ".webp"
}

// copyExif copies EXIF data from source to destination using webp.SetMetadata
func (p *Processor) copyExif(srcPath, destPath string) error {
	// Read source EXIF data
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Extract EXIF data from source
	rawExif, err := exif.SearchAndExtractExifWithReader(srcFile)
	if err != nil {
		// No EXIF data in source, skip
		return nil
	}

	// Read the WebP file
	webpData, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("failed to read webp: %w", err)
	}

	// Set EXIF metadata in WebP
	newData, err := webp.SetMetadata(webpData, rawExif, "EXIF")
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Write back the WebP with EXIF
	if err := os.WriteFile(destPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write webp: %w", err)
	}

	return nil
}
