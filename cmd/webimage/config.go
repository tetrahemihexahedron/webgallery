package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webimage/internal/paths"
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

type config struct {
	inDir   paths.AbsPath
	outDir  paths.AbsPath
	isQuiet bool
	dirDate DirDate
}

func loadConfig() (config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

func parseArgs(args []string, output io.Writer) (config, error) {
	var cfg config
	var inDirPath string
	var outDirPath string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(output)

	flags.StringVar(&inDirPath, "incoming", "", "incoming directory")
	flags.StringVar(&outDirPath, "output", "", "output directory")
	flags.BoolVar(&cfg.isQuiet, "quiet", false, "suppress progress output")

	dirDate := string(DirDateProcessed)
	flags.StringVar(&dirDate, "dir-date", dirDate, "source date for output directories: processed or captured")

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}

	cfg.dirDate = DirDate(dirDate)
	if !cfg.dirDate.isValid() {
		return config{}, fmt.Errorf("invalid '--dir-date' value %q: want %q or %q", dirDate, DirDateProcessed, DirDateCaptured)
	}

	if inDirPath == "" {
		return config{}, errors.New("missing required '--incoming' flag")
	}

	if outDirPath == "" {
		return config{}, errors.New("missing required '--output' flag")
	}

	inDirAbs, err := filepath.Abs(inDirPath)
	if err != nil {
		return config{}, err
	}
	cfg.inDir, err = paths.NewAbsPath(inDirAbs)
	if err != nil {
		return config{}, err
	}

	outDirAbs, err := filepath.Abs(outDirPath)
	if err != nil {
		return config{}, err
	}
	cfg.outDir, err = paths.NewAbsPath(outDirAbs)
	if err != nil {
		return config{}, err
	}

	if err := validateRoots(cfg.inDir, cfg.outDir); err != nil {
		return config{}, err
	}

	return cfg, nil
}

func validateRoots(inDir, outDir paths.AbsPath) error {
	outputRel, err := filepath.Rel(inDir.String(), outDir.String())
	if err != nil {
		return fmt.Errorf("comparing incoming and output directories: %w", err)
	}
	inputRel, err := filepath.Rel(outDir.String(), inDir.String())
	if err != nil {
		return fmt.Errorf("comparing incoming and output directories: %w", err)
	}

	if filepath.IsLocal(outputRel) || filepath.IsLocal(inputRel) {
		return fmt.Errorf("incoming and output directories must not overlap: incoming %q, output %q", inDir, outDir)
	}
	return nil
}
