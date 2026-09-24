package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"tetrahemihexahedron/webgallery/internal/gallery"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := renderGallery(cfg); err != nil {
		log.Fatal(err)
	}
}

func renderGallery(cfg config) error {
	opts := gallery.Options{
		ImagesRoot: cfg.imagesRoot,
		URLPrefix:  cfg.urlPrefix,
		Sort:       cfg.sort,
	}

	var html bytes.Buffer
	if err := gallery.Render(&html, opts); err != nil {
		return fmt.Errorf("rendering gallery: %w", err)
	}

	if cfg.useStdout {
		if _, err := os.Stdout.Write(html.Bytes()); err != nil {
			return fmt.Errorf("writing gallery to stdout: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(cfg.outFile.String(), html.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing output %q: %w", cfg.outFile, err)
	}
	return nil
}
