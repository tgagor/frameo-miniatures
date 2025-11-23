package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/tgagor/frameo-miniatures/internal/discovery"
	"github.com/tgagor/frameo-miniatures/internal/processor"
)

type Config struct {
	InputDir   string
	OutputDir  string
	Resolution string
	Format     string
	Quality    int
	Workers    int
	Prune      bool
	DryRun     bool
	IgnoreFile string
}

func Run(cfg Config) error {
	// Parse resolution
	width, height, err := parseResolution(cfg.Resolution)
	if err != nil {
		return err
	}

	// Setup workers
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Setup processor
	proc := processor.NewProcessor(width, height, cfg.Quality)

	// Setup ignore matcher
	matcher, err := discovery.NewIgnoreMatcher(cfg.IgnoreFile, cfg.InputDir)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load .frameoignore")
		matcher = &discovery.IgnoreMatcher{} // Empty matcher
	}

	// Channels
	files := make(chan discovery.File, 1000)

	// Progress Bar (Indeterminate initially)
	bar := progressbar.NewOptions64(-1,
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	// Start Producer
	go discovery.WalkFiles(cfg.InputDir, files, matcher)

	// Start Consumers
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range files {
				destDir := filepath.Join(cfg.OutputDir, filepath.Dir(file.RelativePath))

				if cfg.DryRun {
					// Simulate
					// time.Sleep(10 * time.Millisecond)
				} else {
					if err := proc.ProcessFile(file.Path, destDir); err != nil {
						log.Error().Err(err).Str("file", file.Path).Msg("Failed to process file")
					}
				}
				bar.Add(1)
			}
		}()
	}

	wg.Wait()
	bar.Finish()

	if cfg.Prune {
		// TODO: Implement Pruning
		log.Info().Msg("Pruning is not yet implemented")
	}

	return nil
}

func parseResolution(res string) (int, int, error) {
	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resolution format: %s", res)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width: %s", parts[0])
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height: %s", parts[1])
	}
	return w, h, nil
}
