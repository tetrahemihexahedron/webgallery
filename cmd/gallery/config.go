package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webimage/internal/gallery"
	"tetrahemihexahedron/webimage/internal/paths"
)

type Config struct {
	ImagesRoot paths.AbsPath
	OutFile    paths.AbsPath
	UseStdout  bool
	URLPrefix  string
	Sort       gallery.SortField
}

func loadConfig() (Config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

func parseArgs(args []string, output io.Writer) (Config, error) {
	var cfg Config
	var imagesRootPath string
	var outFilePath string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(output)

	flags.StringVar(&imagesRootPath, "images", "", "webimage output root containing index.json")
	flags.StringVar(&outFilePath, "output", "", "HTML output file, or stdout when omitted or '-'")
	flags.StringVar(&cfg.URLPrefix, "url-prefix", "", "public URL prefix for image URLs")

	sort := string(gallery.SortCaptured)
	flags.StringVar(&sort, "sort", sort, "sort field: captured or processed")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.Sort = gallery.SortField(sort)
	if !cfg.Sort.IsValid() {
		return Config{}, fmt.Errorf("invalid '--sort' value %q: want %q or %q", sort, gallery.SortCaptured, gallery.SortProcessed)
	}

	if imagesRootPath == "" {
		return Config{}, errors.New("missing required '--images' flag")
	}

	imagesRootAbs, err := filepath.Abs(imagesRootPath)
	if err != nil {
		return Config{}, err
	}
	cfg.ImagesRoot, err = paths.NewAbsPath(imagesRootAbs)
	if err != nil {
		return Config{}, err
	}

	cfg.UseStdout = true
	if outFilePath != "" && outFilePath != "-" {
		outFileAbs, err := filepath.Abs(outFilePath)
		if err != nil {
			return Config{}, err
		}
		cfg.OutFile, err = paths.NewAbsPath(outFileAbs)
		if err != nil {
			return Config{}, err
		}
		cfg.UseStdout = false
	}

	return cfg, nil
}
