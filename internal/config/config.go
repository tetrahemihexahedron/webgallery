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
	InDirAbsPath  string
	OutDirAbsPath string
	IsQuiet       bool
	DirDate       DirDate
}

func Load() (Config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

func parseArgs(args []string, output io.Writer) (Config, error) {
	var cfg Config

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(output)

	flags.StringVar(&cfg.InDirAbsPath, "incoming", "", "incoming directory")
	flags.StringVar(&cfg.OutDirAbsPath, "output", "", "output directory")
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

	if cfg.InDirAbsPath == "" {
		return Config{}, errors.New("missing required '--incoming' flag")
	}

	if cfg.OutDirAbsPath == "" {
		return Config{}, errors.New("missing required '--output' flag")
	}

	inDir, err := filepath.Abs(cfg.InDirAbsPath)
	if err != nil {
		return Config{}, fmt.Errorf("making --incoming path absolute: %w", err)
	}
	cfg.InDirAbsPath = inDir

	outDir, err := filepath.Abs(cfg.OutDirAbsPath)
	if err != nil {
		return Config{}, fmt.Errorf("making --output path absolute: %w", err)
	}
	cfg.OutDirAbsPath = outDir

	return cfg, nil
}
