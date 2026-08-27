package config

import (
	"errors"
	"flag"
	"fmt"
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
	var config Config

	flag.StringVar(&config.InDir, "incoming", "", "incoming directory")
	flag.StringVar(&config.OutDir, "output", "", "output directory")
	flag.BoolVar(&config.IsQuiet, "quiet", false, "suppress progress output")

	dirDate := string(DirDateProcessed)
	flag.StringVar(&dirDate, "dir-date", dirDate, "source date for output directories: processed or captured")

	flag.Parse()

	config.DirDate = DirDate(dirDate)
	if !config.DirDate.isValid() {
		return Config{}, fmt.Errorf("invalid '--dir-date' value %q: want %q or %q", dirDate, DirDateProcessed, DirDateCaptured)
	}

	if config.InDir == "" {
		return Config{}, errors.New("missing required '--incoming' flag")
	}

	if config.OutDir == "" {
		return Config{}, errors.New("missing required '--output' flag")
	}

	return config, nil
}
