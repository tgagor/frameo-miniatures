# Project Specification: Frameo Miniatures

## 1. Overview
**Frameo Miniatures** is a CLI application written in Go designed to prepare and optimize a photo collection for Frameo digital photo frames.
Frameo frames often have specific resolution limits and limited storage. Modern cameras produce huge files (HEIC/JPG) that are inefficient for these devices.
This tool scans a source directory, resizes images to fit the frame's resolution (preserving aspect ratio), converts them to a space-efficient format (WebP), and handles file system constraints (FAT32), all while preserving the directory structure and critical metadata (EXIF dates).

## 2. Technical Stack
- **Language:** Go (Golang) 1.23+
- **CLI Framework:** `github.com/spf13/cobra`
- **Logging:** `github.com/rs/zerolog`
- **Image Processing:**
  - Standard `image` package.
  - `golang.org/x/image/draw` for high-quality resizing (Catmull-Rom or Bi-Linear).
  - `github.com/chai2010/webp` (or similar) for WebP encoding.
  - **HEIC Support:** `github.com/adrium/goheif` (Note: May require CGO/libheif. If strictly pure Go is needed, this feature may need to be optional or use a different library).
- **Progress Bar:** `github.com/schollz/progressbar/v3` or `github.com/vbauerster/mpb`.
- **Configuration:** `spf13/viper` (optional, if config file needed) or just Cobra flags.

## 3. Functional Requirements

### 3.1 Input & Discovery
- **Recursive Search:** Walk through the `--input` directory.
- **Streaming Discovery (Nice to Have):**
  - To minimize startup time on large libraries (e.g., 100k files), file discovery should run concurrently with processing.
  - As valid files are found, they should be immediately sent to the processing queue (channel).
  - *Note:* This implies the total file count for the progress bar will update dynamically or be unknown initially.
- **Allowed Extensions:** `.jpg`, `.jpeg`, `.heic` (Case insensitive).
- **Ignore Logic:**
  - Look for a `.frameoignore` file in the root of the input directory.
  - Syntax: Same as `.gitignore` (glob patterns).
  - Files or directories matching these patterns must be skipped.

### 3.2 Image Processing Pipeline
For each valid image found:
1.  **Decode:** Read the image. Handle HEIC decoding if possible.
3.  **Auto-Rotate:**
    - Apply orientation correction based on EXIF metadata (e.g., if the image is rotated 90 degrees, physically rotate the pixels).
4.  **Resize:**
    - Target resolution defined by `--resolution` (e.g., `1280x800`).
    - **Logic:** "Fit Within" (Contain). The image should be resized so that it fits within the specified bounding box (width x height), preserving aspect ratio. Do not crop.
    - The `resolution` argument defines the maximum dimensions. For example, a portrait photo in a `1280x800` frame context should be resized to fit within `800x1280` (if the frame can rotate) or just fit within the bounding box provided. *Clarification:* Treat the provided resolution as the maximum bounding box for the image.
5.  **Normalize Filename:**
    - Output filesystem is likely FAT32 (SD cards).
    - Remove/Replace invalid characters: `\ / : * ? " < > |`.
    - Ensure filename length is within limits (though usually fine).
6.  **Preserve Metadata:**
    - **Crucial:** Frameo uses EXIF "DateTimeOriginal" or "CreateDate" to sort photos.
    - The tool **MUST** copy these EXIF tags from the source to the destination file.
    - If EXIF is missing, attempt to set the file's modification time to match the source.
7.  **Encode:**
    - Format: Defaults to `.webp` (configurable).
    - Quality: Defaults to `80` (configurable).
8.  **Save:**
    - Write to the `--output` directory, mirroring the relative path from the input.

### 3.3 Synchronization & Pruning
- **Structure Mirroring:** Create subdirectories in output as needed.
- **Pruning (`--prune`):**
  - If enabled, after processing, scan the output directory.
  - Remove any files that:
    - Do not have a corresponding source file in the input (accounting for extension change).
    - OR correspond to a source file that is now ignored via `.frameoignore`.

### 3.4 Performance & UX
- **Concurrency:**
  - Use a **Producer-Consumer** pattern.
  - **Producer:** A goroutine walks the directory tree and feeds file paths into a buffered channel.
  - **Consumer:** A pool of worker goroutines (default `runtime.NumCPU()`) reads from the channel and processes images.
- **Progress:** Display a progress bar showing:
  - Completed / Total files.
  - Percentage.
  - ETA (Estimated Time of Arrival).
- **Logging:** Use `zerolog`.
  - Normal operation: Info/Error logs.
  - Pretty printing for console output.

## 4. CLI Interface

### Command Structure
`frameo-miniatures [command] [flags]`

### Root Flags
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--input` | `-i` | string | `.` | Source directory path. |
| `--output` | `-o` | string | `./output` | Destination directory path. |
| `--resolution` | `-r` | string | `1280x800` | Target frame resolution (bounding box). |
| `--format` | `-f` | string | `webp` | Output format (`webp`, `jpg`). |
| `--quality` | `-q` | int | `80` | Compression quality (0-100). |
| `--workers` | `-j` | int | `0` | Number of concurrent workers (0 = auto). |
| `--prune` | | bool | `false` | Delete files in output that are not in input. |
| `--dry-run` | | bool | `false` | Simulate without writing files. |

## 5. Development Plan (Agent Instructions)

**Phase 1: Project Skeleton & CLI**
- Initialize module `go mod init`.
- Setup Cobra root command and flags.
- Setup Zerolog.

**Phase 2: File Discovery & Ignore System**
- Implement `WalkDir` logic.
- Parse `.frameoignore` (can use a library like `github.com/sabhiram/go-gitignore`).
- List all valid files to be processed.

**Phase 3: Image Processing Core**
- Implement `Processor` struct.
- Handle Loading -> Resizing -> Encoding.
- Implement Metadata copying (check `github.com/dsoprea/go-exif` or similar if standard libs fail).

**Phase 4: Concurrency & Progress**
- Implement worker pool.
- Hook up progress bar.

**Phase 5: Pruning & Polish**
- Implement the cleanup logic.
- Final testing with edge cases (bad files, weird names).
