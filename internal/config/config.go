package config

import (
	"errors"
	"flag"
)

type Config struct {
	InDir   string
	OutDir  string
	IsQuiet bool
}

func Load() (Config, error) {
	var config Config

	flag.StringVar(&config.InDir, "incoming", "", "incoming directory")
	flag.StringVar(&config.OutDir, "output", "", "output directory")
	flag.BoolVar(&config.IsQuiet, "quiet", false, "suppress progress output")

	flag.Parse()

	if config.InDir == "" {
		return Config{}, errors.New("missing required '--incoming' flag")
	}

	if config.OutDir == "" {
		return Config{}, errors.New("missing required '--output' flag")
	}

	return config, nil
}
