package main

import (
	"io"
	"log"
	"os"

	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	progressReporter := io.Writer(os.Stdout)
	if cfg.IsQuiet {
		progressReporter = io.Discard
	}

	processor := processor{
		cfg:              cfg,
		metadataReader:   &metadata.Exiftool{},
		variantGenerator: &variants.Vipsthumbnail{},
		progressReporter: progressReporter,
	}

	_, err = processor.processDir()
	if err != nil {
		log.Fatal(err)
	}
}
