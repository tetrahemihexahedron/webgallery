package main

import (
	"fmt"
	"io"
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

func renderGallery(cfg Config) (err error) {
	writer := io.Writer(os.Stdout)

	if !cfg.UseStdout {
		file, err := os.Create(cfg.OutFile.String())
		if err != nil {
			return fmt.Errorf("opening output %q: %w", cfg.OutFile, err)
		}
		defer func() {
			if closeErr := file.Close(); err == nil && closeErr != nil {
				err = fmt.Errorf("closing output %q: %w", cfg.OutFile, closeErr)
			}
		}()

		writer = file
	}

	opts := gallery.Options{
		ImagesRoot: cfg.ImagesRoot,
		URLPrefix:  cfg.URLPrefix,
		Sort:       cfg.Sort,
	}
	if err := gallery.Render(writer, opts); err != nil {
		return fmt.Errorf("rendering gallery: %w", err)
	}

	return nil
}
