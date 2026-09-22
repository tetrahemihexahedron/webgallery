package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	progressReporter := io.Writer(os.Stdout)
	if cfg.isQuiet {
		progressReporter = io.Discard
	}

	processor := imageProcessor{
		cfg:              cfg,
		metadataReader:   metadata.Read,
		variantGenerator: variants.Generate,
		progressReporter: progressReporter,
	}

	_, err = processor.processIncomingDir(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
