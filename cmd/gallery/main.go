package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"tetrahemihexahedron/webimage/internal/gallery"
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

func renderGallery(cfg Config) error {
	opts := gallery.Options{
		ImagesRoot: cfg.ImagesRoot,
		URLPrefix:  cfg.URLPrefix,
		Sort:       cfg.Sort,
	}

	var html bytes.Buffer
	if err := gallery.Render(&html, opts); err != nil {
		return fmt.Errorf("rendering gallery: %w", err)
	}

	if cfg.UseStdout {
		if _, err := os.Stdout.Write(html.Bytes()); err != nil {
			return fmt.Errorf("writing gallery to stdout: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(cfg.OutFile.String(), html.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing output %q: %w", cfg.OutFile, err)
	}
	return nil
}
