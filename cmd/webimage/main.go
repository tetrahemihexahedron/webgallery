package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

func main() {
	cfg, err := config.Load()
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

	result, err := processor.processDir()
	if err != nil {
		log.Fatal(err)
	}

	for _, image := range result.images {
		fmt.Printf("%s: %d variants\n", image.Dir, len(image.Variants))
	}

	for _, problem := range result.problems {
		fmt.Printf("%s: %s\n", problem.fileName, problem.message)
	}
}
