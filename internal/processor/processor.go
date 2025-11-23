package processor

import (
	"fmt"
	"image"
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
}

// NewProcessor creates a new processor
func NewProcessor(width, height, quality int) *Processor {
	return &Processor{
		Width:   width,
		Height:  height,
		Quality: quality,
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

	// 6. Encode (WebP)
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	if err := webp.Encode(out, img, &webp.Options{Quality: float32(p.Quality)}); err != nil {
		return fmt.Errorf("failed to encode webp: %w", err)
	}

	// 7. Preserve Metadata (Time)
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

	return nameWithoutExt + ".webp"
}
