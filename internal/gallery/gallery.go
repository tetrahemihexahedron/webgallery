package gallery

import (
	"errors"
	"fmt"
	"io"

	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/paths"
)

// SortField selects which image datetime field controls display order.
type SortField string

const (
	SortCaptured  SortField = "captured"
	SortProcessed SortField = "processed"
)

// IsValid reports whether s is a supported gallery sort field.
func (s SortField) IsValid() bool {
	switch s {
	case SortCaptured, SortProcessed:
		return true
	default:
		return false
	}
}

// Options configures gallery rendering.
type Options struct {
	ImagesRoot paths.AbsPath
	URLPrefix  string
	Sort       SortField
}

type galleryImage struct {
	indexImage index.Image
	manifest   manifest.Manifest
}

// Render writes gallery HTML to w.
func Render(w io.Writer, opts Options) error {
	if w == nil {
		return errors.New("writer can't be nil")
	}
	if !opts.Sort.IsValid() {
		return fmt.Errorf("invalid sort field %q", opts.Sort)
	}

	images, err := loadImages(opts)
	if err != nil {
		return err
	}
	_ = images

	return errors.New("gallery rendering is not implemented")
}

func loadImages(opts Options) ([]galleryImage, error) {
	idx, err := index.Read(opts.ImagesRoot)
	if err != nil {
		return nil, fmt.Errorf("reading gallery index: %w", err)
	}

	images := make([]galleryImage, 0, len(idx.Images))
	for _, indexImage := range idx.Images {
		manifestPath, err := paths.JoinAbs(opts.ImagesRoot, indexImage.Manifest)
		if err != nil {
			return nil, fmt.Errorf("image %q: building manifest path %q: %w", indexImage.Dir, indexImage.Manifest, err)
		}

		mani, err := manifest.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("image %q: reading manifest %q: %w", indexImage.Dir, indexImage.Manifest, err)
		}

		images = append(images, galleryImage{
			indexImage: indexImage,
			manifest:   mani,
		})
	}

	return images, nil
}
