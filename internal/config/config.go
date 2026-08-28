package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type DirDate string

const (
	DirDateProcessed DirDate = "processed"
	DirDateCaptured  DirDate = "captured"
)

func (d DirDate) isValid() bool {
	switch d {
	case DirDateProcessed, DirDateCaptured:
		return true
	default:
		return false
	}
}

type Config struct {
	InDir   string
	OutDir  string
	IsQuiet bool
	DirDate DirDate
}

func Load() (Config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

func parseArgs(args []string, output io.Writer) (Config, error) {
	var cfg Config

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(output)

	flags.StringVar(&cfg.InDir, "incoming", "", "incoming directory")
	flags.StringVar(&cfg.OutDir, "output", "", "output directory")
	flags.BoolVar(&cfg.IsQuiet, "quiet", false, "suppress progress output")

	dirDate := string(DirDateProcessed)
	flags.StringVar(&dirDate, "dir-date", dirDate, "source date for output directories: processed or captured")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.DirDate = DirDate(dirDate)
	if !cfg.DirDate.isValid() {
		return Config{}, fmt.Errorf("invalid '--dir-date' value %q: want %q or %q", dirDate, DirDateProcessed, DirDateCaptured)
	}

	if cfg.InDir == "" {
		return Config{}, errors.New("missing required '--incoming' flag")
	}

	if cfg.OutDir == "" {
		return Config{}, errors.New("missing required '--output' flag")
	}

	inDir, err := filepath.Abs(cfg.InDir)
	if err != nil {
		return Config{}, fmt.Errorf("making --incoming path absolute: %w", err)
	}
	cfg.InDir = inDir

	outDir, err := filepath.Abs(cfg.OutDir)
	if err != nil {
		return Config{}, fmt.Errorf("making --output path absolute: %w", err)
	}
	cfg.OutDir = outDir

	return cfg, nil
}
