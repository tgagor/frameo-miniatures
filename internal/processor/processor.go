package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrium/goheif"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/rs/zerolog/log"
	"github.com/tgagor/frameo-miniatures/internal/fileutil"
)

// Processor handles image processing
type Processor struct {
	Width        int
	Height       int
	Quality      int
	Format       string // "webp" or "jpg"
	SkipExisting bool
}

// NewProcessor creates a new processor
func NewProcessor(width, height, quality int, format string, skipExisting bool) *Processor {
	return &Processor{
		Width:        width,
		Height:       height,
		Quality:      quality,
		Format:       format,
		SkipExisting: skipExisting,
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
	// Determine target dimensions based on orientation
	// We want to optimize for the frame's resolution regardless of its current orientation.
	// So we define the frame's "Long" and "Short" dimensions.
	frameLong := p.Width
	if p.Height > frameLong {
		frameLong = p.Height
	}
	frameShort := p.Width
	if p.Height < frameShort {
		frameShort = p.Height
	}

	// Check image orientation
	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()

	var targetW, targetH int
	if imgW >= imgH {
		// Landscape image: Fit into Frame Landscape (Long x Short)
		targetW = frameLong
		targetH = frameShort
	} else {
		// Portrait image: Fit into Frame Portrait (Short x Long)
		targetW = frameShort
		targetH = frameLong
	}

	// "Fit Within" - imaging.Fit keeps aspect ratio
	img = imaging.Fit(img, targetW, targetH, imaging.CatmullRom)

	// 5. Normalize Filename
	destFilename := p.normalizeFilename(filepath.Base(srcPath))
	destPath := filepath.Join(destDir, destFilename)

	// Check if file exists if SkipExisting is enabled
	if p.SkipExisting {
		if _, err := os.Stat(destPath); err == nil {
			// File exists, skip
			return nil
		}
	}

	// Ensure dest dir exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create dest dir: %w", err)
	}

	// 6. Encode to memory buffer first
	var buf bytes.Buffer

	// Encode based on format
	if p.Format == "jpg" || p.Format == "jpeg" {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: p.Quality})
		if err != nil {
			return fmt.Errorf("failed to encode jpeg: %w", err)
		}
	} else {
		// Default to WebP
		err = webp.Encode(&buf, img, &webp.Options{Quality: float32(p.Quality)})
		if err != nil {
			return fmt.Errorf("failed to encode webp: %w", err)
		}
	}

	// 7. Add EXIF metadata to encoded data (before writing to disk)
	encodedData := buf.Bytes()

	// Reset file pointer for EXIF extraction
	f.Seek(0, 0)
	rawExif, err = exif.SearchAndExtractExifWithReader(f)
	if err == nil {
		// Rebuild EXIF with only allowed tags
		rebuiltExif, err := p.rebuildExif(rawExif)
		if err != nil {
			log.Warn().Err(err).Str("src", srcPath).Msg("Failed to rebuild EXIF, skipping metadata")
			// If rebuild fails, we skip EXIF entirely to avoid embedding broken/large data
		} else {
			rawExif = rebuiltExif

			// We have EXIF data, embed it
			switch p.Format {
			case "webp":
				// For WebP, use SetMetadata
				encodedData, err = webp.SetMetadata(encodedData, rawExif, "EXIF")
				if err != nil {
					log.Warn().Err(err).Str("src", srcPath).Msg("Failed to embed EXIF in WebP")
				}
			case "jpg", "jpeg":
				// For JPEG, use go-jpeg-image-structure
				encodedData, err = p.embedExifInJPEG(encodedData, rawExif)
				if err != nil {
					log.Warn().Err(err).Str("src", srcPath).Msg("Failed to embed EXIF in JPEG")
				}
			}
		}
	}

	// 8. Write final data to disk (single write operation)
	if err := os.WriteFile(destPath, encodedData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// 9. Set file modification time
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
	return fileutil.GetOutputFilename(name, p.Format)
}

// embedExifInJPEG embeds EXIF data into JPEG bytes
func (p *Processor) embedExifInJPEG(jpegData, exifData []byte) ([]byte, error) {
	// Parse the JPEG structure
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(jpegData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JPEG: %w", err)
	}

	sl := intfc.(*jpegstructure.SegmentList)

	// Construct EXIF builder from raw EXIF data
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, fmt.Errorf("failed to create IFD mapping: %w", err)
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, exifData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EXIF data: %w", err)
	}

	// Create IfdBuilder from the root IFD
	ib, err := safeNewIfdBuilderFromExistingChain(index.RootIfd)
	if err != nil {
		return nil, fmt.Errorf("failed to create IFD builder: %w", err)
	}

	// Set the EXIF data
	err = sl.SetExif(ib)
	if err != nil {
		return nil, fmt.Errorf("failed to set EXIF: %w", err)
	}

	// Write the updated JPEG to a buffer
	var buf bytes.Buffer
	err = sl.Write(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// rebuildExif creates a new EXIF block with only allowed tags
func (p *Processor) rebuildExif(rawExif []byte) ([]byte, error) {
	// Parse all tags
	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {
		return nil, err
	}

	// Create new builder
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, err
	}
	ti := exif.NewTagIndex()

	// Default to BigEndian as it's common for EXIF
	ib := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)

	// Allowed tags
	allowedTags := map[string]bool{
		"DateTime":            true,
		"DateTimeOriginal":    true,
		"CreateDate":          true,
		"OffsetTime":          true,
		"OffsetTimeOriginal":  true,
		"OffsetTimeDigitized": true,
		"Make":                true,
		"Model":               true,
		"GPSLatitude":         true,
		"GPSLongitude":        true,
		"GPSAltitude":         true,
		"GPSDateStamp":        true,
		"GPSTimeStamp":        true,
		"GPSProcessingMethod": true,
		"GPSAreaInformation":  true,
	}

	for _, tag := range entries {
		// Skip if not allowed
		if !allowedTags[tag.TagName] {
			continue
		}

		// Skip Orientation explicitly as we rotate the image
		if tag.TagName == "Orientation" {
			continue
		}

		// Add to builder
		// We use AddStandardWithName which handles looking up the tag ID
		// tag.IfdPath gives us the hierarchy (e.g. "IFD0", "IFD/Exif", "IFD/GPS")
		// AddStandardWithName(name string, value interface{}) error -> It seems it doesn't take IfdPath?
		// Wait, if I use the root builder `ib`, how does it know where to put it?
		// Ah, `AddStandardWithName` is a method on `IfdBuilder`.
		// If the tag belongs to a child IFD (like Exif or GPS), we need to get/create that child builder first.

		targetIb, err := exif.GetOrCreateIbFromRootIb(ib, tag.IfdPath)
		if err != nil {
			log.Debug().Err(err).Str("path", tag.IfdPath).Msg("Failed to get/create IFD builder")
			continue
		}

		err = targetIb.AddStandardWithName(tag.TagName, tag.Value)
		if err != nil {
			// Log but continue? Or fail?
			// Some tags might fail to add if value type doesn't match what standard expects.
			// Given we want to be robust, we should probably ignore errors for individual tags.
			log.Debug().Err(err).Str("tag", tag.TagName).Msg("Failed to add tag to new EXIF")
		}
	}

	// Encode
	ibe := exif.NewIfdByteEncoder()
	return ibe.EncodeToExif(ib)
}

// safeNewIfdBuilderFromExistingChain wraps exif.NewIfdBuilderFromExistingChain to recover from panics
func safeNewIfdBuilderFromExistingChain(rootIfd *exif.Ifd) (ib *exif.IfdBuilder, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in NewIfdBuilderFromExistingChain: %v", r)
		}
	}()
	ib = exif.NewIfdBuilderFromExistingChain(rootIfd)
	return ib, nil
}
