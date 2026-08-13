package main

import (
	"fmt"
	"log"
	"tetrahemihexahedron/webimage/internal/config"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processing directory %s\n", config.InDir)

	result, err := ProcessDir(config)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	for _, image := range result.Images {
		fmt.Printf("%s: %d variants\n", image.Dir, len(image.Variants))
	}

	for _, problem := range result.Problems {
		fmt.Printf("%s: %s\n", problem.FileName, problem.Message)
	}
}
