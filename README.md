# Frameo Miniatures

A fast, efficient CLI tool to prepare and optimize photo collections for Frameo digital photo frames.

## Overview

Frameo Miniatures resizes, compresses, and organizes your photos to fit perfectly on Frameo digital frames. Modern cameras produce huge files that are inefficient for photo frames with limited resolution and storage. This tool:

- **Resizes** images to fit your frame's resolution while preserving aspect ratio
- **Converts** to space-efficient WebP format (or JPEG)
- **Preserves** EXIF metadata (capture dates, orientation)
- **Auto-rotates** images based on EXIF orientation data
- **Mirrors** your directory structure
- **Handles** FAT32 filename constraints
- **Processes** files in parallel for maximum speed

## Features

- ✅ Supports JPG, JPEG, and HEIC formats
- ✅ Streaming file discovery (starts processing immediately)
- ✅ Configurable ignore patterns (`.frameoignore`)
- ✅ Progress bar with ETA
- ✅ Dry-run mode for testing
- ✅ Pruning of outdated files
- ✅ Skip existing files for fast incremental updates
- ✅ Multi-core processing

## Installation

### From Source

```bash
git clone https://github.com/tgagor/frameo-miniatures.git
cd frameo-miniatures
make build
sudo make install
```

### Using Go

```bash
go install github.com/tgagor/frameo-miniatures@latest
```

## Quick Start

Basic usage:

```bash
frameo-miniatures -i ./photos -o ./miniatures
```

This will:
1. Scan `./photos` for images
2. Resize them to fit 1280x800 (default)
3. Convert to WebP format
4. Save to `./miniatures` preserving directory structure

## Usage

```
frameo-miniatures [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input` | `-i` | `.` | Source directory path |
| `--output` | `-o` | `./output` | Destination directory path |
| `--resolution` | `-r` | `1280x800` | Target frame resolution (bounding box) |
| `--format` | `-f` | `webp` | Output format (`webp`, `jpg`) |
| `--quality` | `-q` | `80` | Compression quality (0-100) |
| `--workers` | `-j` | `0` | Number of concurrent workers (0 = auto) |
| `--ignore-file` | | | Path to custom `.frameoignore` file |
| `--prune` | | `false` | Remove orphaned files from output (no source or ignored) |
| `--skip-existing` | | `false` | Skip processing if output file already exists |
| `--dry-run` | | `false` | Simulate without writing files |
| `--version` | | | Show version information |

### Examples

**Process photos for a 1920x1080 frame:**
```bash
frameo-miniatures -i ~/Photos -o ~/FramePhotos -r 1920x1080
```

**Use JPEG instead of WebP:**
```bash
frameo-miniatures -i ./photos -o ./miniatures -f jpg -q 85
```

**Dry run to see what would happen:**
```bash
frameo-miniatures -i ./photos -o ./miniatures --dry-run
```

**Prune orphaned files from output:**
```bash
frameo-miniatures -i ./photos -o ./miniatures --prune
```
This removes miniatures that no longer have corresponding source files or match ignore patterns.

**Use custom ignore file:**
```bash
frameo-miniatures -i ./photos -o ./miniatures --ignore-file ~/my-ignore-rules
```

**Skip existing files (incremental update):**
```bash
frameo-miniatures -i ./photos -o ./miniatures --skip-existing
```

**Incremental update with cleanup:**
```bash
frameo-miniatures -i ./photos -o ./miniatures --skip-existing --prune
```
This efficiently updates only new/changed files and removes orphaned miniatures.

## Ignore Patterns

You can exclude files and directories using a `.frameoignore` file. The syntax is similar to `.gitignore`.

### Search Order

The tool looks for `.frameoignore` in the following order (unless `--ignore-file` is specified):

1. `--ignore-file` path (if provided)
2. `~/.config/frameoignore`
3. Input directory
4. Current directory

### Example `.frameoignore`

```
# Ignore all files in specific directories
*/2005.07/Ognisko u Gogusia/*
2002.03/Studniówka z Beatą/*

# Ignore by pattern
*.tmp
*.bak
*_draft*

# Ignore specific directories
temp/
drafts/
```

### Pattern Syntax

- `*` matches any characters except `/`
- `**` matches any characters including `/`
- `!` negates a pattern
- Lines starting with `#` are comments

## How It Works

1. **Discovery**: Walks the input directory recursively, finding valid image files
2. **Filtering**: Applies `.frameoignore` rules to skip unwanted files
3. **Processing**: For each image:
   - Decodes the image (JPG/HEIC)
   - Reads EXIF metadata
   - Auto-rotates based on EXIF orientation
   - Resizes to fit within target resolution (preserving aspect ratio)
   - Normalizes filename for FAT32 compatibility
   - Encodes to WebP (or JPEG)
   - Preserves capture date/time
4. **Pruning** (optional): Removes orphaned miniatures from output directory
   - Deletes files with no corresponding source
   - Removes files matching ignore patterns

## Performance

The tool uses a producer-consumer pattern with parallel processing:

- **Producer**: Walks directories and streams files to a queue
- **Consumers**: Multiple workers process images concurrently
- **Default**: Uses all CPU cores for maximum speed

On a typical system, you can expect:
- ~10-50 images/second (depending on size and format)
- Immediate start (streaming discovery)
- Linear scaling with CPU cores

## Technical Details

### Supported Formats

**Input:**
- JPEG (`.jpg`, `.jpeg`)
- HEIC (`.heic`)

**Output:**
- WebP (default, best compression)
- JPEG

### Image Processing

- **Resizing**: Catmull-Rom resampling for high quality
- **Aspect Ratio**: Always preserved
- **Orientation**: Auto-corrected from EXIF
- **Metadata**: EXIF dates copied to output files

### File System

- **Filename Normalization**: Removes invalid FAT32 characters (`\ / : * ? " < > |`)
- **Directory Structure**: Mirrored from input to output
- **Extension**: Changed to match output format (e.g., `.jpg` → `.webp`)

## Troubleshooting

### Files not being ignored

Make sure your `.frameoignore` patterns are correct:
- Use `*/dirname/*` for directories with a parent
- Use `dirname/*` for top-level directories
- Check the log output to see which ignore file was loaded

### HEIC support issues

HEIC decoding requires CGO and libheif. If you encounter issues:
- Ensure libheif is installed on your system
- Build with CGO enabled: `CGO_ENABLED=1 go build`

### Performance issues

- Reduce `--workers` if system is overloaded
- Use `--dry-run` to test without writing files
- Check disk I/O (especially on network drives)

## License

[Add your license here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

[Add your name/info here]
