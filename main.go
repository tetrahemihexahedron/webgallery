package main

import (
	"log"
	"tetrahemihexahedron/webimage/internal/config"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Loaded config: %+v", config)

	ProcessDir(config)
}
