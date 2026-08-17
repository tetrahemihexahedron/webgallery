package main

import (
	"fmt"
	"log"

	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processing directory %s\n", cfg.InDir)

	processor := processor{
		cfg:              cfg,
		metadataReader:   &metadata.Exiftool{},
		variantGenerator: &variants.Vipsthumbnail{},
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
