package main

import (
	"fmt"
	"log"
	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processing directory %s\n", config.InDir)

	processor := processor{
		cfg:              config,
		metadataReader:   &metadata.Exiftool{},
		variantGenerator: &variants.Vipsthumbnail{},
	}

	result, err := processor.ProcessDir()
	if err != nil {
		log.Fatal(err)
	}

	for _, image := range result.Images {
		fmt.Printf("%s: %d variants\n", image.Dir, len(image.Variants))
	}

	for _, problem := range result.Problems {
		fmt.Printf("%s: %s\n", problem.FileName, problem.Message)
	}
}
