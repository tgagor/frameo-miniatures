package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tgagor/frameo-miniatures/internal/app"
)

var (
	inputDir   string
	outputDir  string
	resolution string
	format     string
	quality    int
	workers    int
	prune      bool
	dryRun     bool
	ignoreFile string
)

var rootCmd = &cobra.Command{
	Use:   "frameo-miniatures",
	Short: "Prepare and optimize photos for Frameo digital frames",
	Long: `Frameo Miniatures is a CLI tool to resize, compress, and organize photos
for Frameo digital photo frames. It supports resizing with aspect ratio preservation,
WebP conversion, and metadata copying.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Setup logging
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().
			Str("input", inputDir).
			Str("output", outputDir).
			Str("resolution", resolution).
			Str("format", format).
			Int("quality", quality).
			Int("workers", workers).
			Bool("prune", prune).
			Bool("dry_run", dryRun).
			Msg("Starting Frameo Miniatures")

		cfg := app.Config{
			InputDir:   inputDir,
			OutputDir:  outputDir,
			Resolution: resolution,
			Format:     format,
			Quality:    quality,
			Workers:    workers,
			Prune:      prune,
			DryRun:     dryRun,
			IgnoreFile: ignoreFile,
		}

		if err := app.Run(cfg); err != nil {
			log.Fatal().Err(err).Msg("Application failed")
		}
	},
}

func Execute(appName string, version string) {
	rootCmd.Use = appName
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&inputDir, "input", "i", ".", "Source directory path")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "Destination directory path")
	rootCmd.Flags().StringVarP(&resolution, "resolution", "r", "1280x800", "Target frame resolution (bounding box)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "webp", "Output format (webp, jpg)")
	rootCmd.Flags().IntVarP(&quality, "quality", "q", 80, "Compression quality (0-100)")
	rootCmd.Flags().IntVarP(&workers, "workers", "j", 0, "Number of concurrent workers (0 = auto)")
	rootCmd.Flags().BoolVar(&prune, "prune", false, "Delete files in output that are not in input")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate without writing files")
	rootCmd.Flags().StringVar(&ignoreFile, "ignore-file", "", "Path to .frameoignore file")
}
