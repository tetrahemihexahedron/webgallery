package gallery

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"

	"tetrahemihexahedron/webimage/internal/image"
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
	if err := sortImages(images, opts.Sort); err != nil {
		return err
	}
	_ = images

	return errors.New("gallery rendering is not implemented")
}

func sortImages(images []galleryImage, sortField SortField) error {
	if err := validateSortValues(images, sortField); err != nil {
		return err
	}

	slices.SortFunc(images, func(a, b galleryImage) int {
		aValue := sortValue(a, sortField)
		bValue := sortValue(b, sortField)
		if aValue != bValue {
			return cmp.Compare(bValue, aValue)
		}
		// use directory name as tie-breaker
		return cmp.Compare(a.indexImage.Dir.String(), b.indexImage.Dir.String())
	})

	return nil
}

func validateSortValues(images []galleryImage, sortField SortField) error {
	for _, img := range images {
		if err := parseSortValue(img, sortField); err != nil {
			return fmt.Errorf("image %q: %w", img.indexImage.Dir, err)
		}
	}

	return nil
}

func parseSortValue(img galleryImage, sortField SortField) error {
	switch sortField {
	case SortCaptured:
		if _, err := image.ParseCapturedAt(img.indexImage.CapturedAt); err != nil {
			return fmt.Errorf("invalid capturedAt: %w", err)
		}
		return nil
	case SortProcessed:
		if _, err := image.ParseProcessedAt(img.indexImage.ProcessedAt); err != nil {
			return fmt.Errorf("invalid processedAt: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("invalid sort field %q", sortField)
	}
}

func sortValue(img galleryImage, sortField SortField) string {
	switch sortField {
	case SortCaptured:
		return img.indexImage.CapturedAt
	case SortProcessed:
		return img.indexImage.ProcessedAt
	default:
		return ""
	}
}

func publicURL(urlPrefix string, imgDir paths.RelPath, file paths.RelPath) string {
	return path.Join(urlPrefix, imgDir.String(), file.String())
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
