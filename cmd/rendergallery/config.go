package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webgallery/internal/gallery"
	"tetrahemihexahedron/webgallery/internal/paths"
)

type config struct {
	imagesRoot paths.AbsPath
	outFile    paths.AbsPath
	useStdout  bool
	urlPrefix  string
	sort       gallery.SortField
}

func loadConfig() (config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

func parseArgs(args []string, output io.Writer) (config, error) {
	var cfg config
	var imagesRootPath string
	var outFilePath string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(output)

	flags.StringVar(&imagesRootPath, "images", "", "prepgallery output root containing index.json")
	flags.StringVar(&outFilePath, "output", "", "HTML output file or stdout when omitted or '-'")
	flags.StringVar(&cfg.urlPrefix, "url-prefix", "", "public URL prefix for image URLs")

	sort := string(gallery.SortProcessed)
	flags.StringVar(&sort, "sort", sort, "sort field: processed (default) or captured (requires capturedAt for every image)")

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}

	sortField, err := gallery.ParseSortField(sort)
	if err != nil {
		return config{}, fmt.Errorf("parsing '--sort': %w", err)
	}
	cfg.sort = sortField

	if imagesRootPath == "" {
		return config{}, errors.New("missing required '--images' flag")
	}

	imagesRootAbs, err := filepath.Abs(imagesRootPath)
	if err != nil {
		return config{}, err
	}
	cfg.imagesRoot, err = paths.NewAbsPath(imagesRootAbs)
	if err != nil {
		return config{}, err
	}

	cfg.useStdout = true
	if outFilePath != "" && outFilePath != "-" {
		outFileAbs, err := filepath.Abs(outFilePath)
		if err != nil {
			return config{}, err
		}
		cfg.outFile, err = paths.NewAbsPath(outFileAbs)
		if err != nil {
			return config{}, err
		}
		cfg.useStdout = false
	}

	return cfg, nil
}
