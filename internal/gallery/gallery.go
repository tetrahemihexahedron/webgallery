package gallery

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

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

// ParseSortField parses a supported gallery sort field.
func ParseSortField(s string) (SortField, error) {
	field := SortField(s)
	if !field.IsValid() {
		return "", fmt.Errorf("invalid sort field %q: want %q or %q", s, SortCaptured, SortProcessed)
	}
	return field, nil
}

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

const (
	imageSizes           = "(max-width: 700px) 100vw, 700px"
	fallbackDisplayWidth = 700
)

type loadedImage struct {
	indexEntry index.Entry
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
	data, err := newTemplateData(images, opts.URLPrefix)
	if err != nil {
		return err
	}

	return renderHTML(w, data)
}

func renderHTML(w io.Writer, data templateData) error {
	if err := galleryTemplate.Execute(w, data); err != nil {
		return fmt.Errorf("rendering gallery template: %w", err)
	}
	return nil
}

func sortImages(images []loadedImage, sortField SortField) error {
	if err := validateSortValues(images, sortField); err != nil {
		return err
	}

	slices.SortFunc(images, func(a, b loadedImage) int {
		aValue := sortValue(a, sortField)
		bValue := sortValue(b, sortField)
		if aValue != bValue {
			return cmp.Compare(bValue, aValue)
		}
		// use directory name as tie-breaker
		return cmp.Compare(a.indexEntry.Dir.String(), b.indexEntry.Dir.String())
	})

	return nil
}

func validateSortValues(images []loadedImage, sortField SortField) error {
	for _, img := range images {
		if err := parseSortValue(img, sortField); err != nil {
			return fmt.Errorf("image %q: %w", img.indexEntry.Dir, err)
		}
	}

	return nil
}

func parseSortValue(img loadedImage, sortField SortField) error {
	switch sortField {
	case SortCaptured:
		if _, err := image.ParseCapturedAt(img.indexEntry.CapturedAt); err != nil {
			return fmt.Errorf("invalid capturedAt: %w", err)
		}
		return nil
	case SortProcessed:
		if _, err := image.ParseProcessedAt(img.indexEntry.ProcessedAt); err != nil {
			return fmt.Errorf("invalid processedAt: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("invalid sort field %q", sortField)
	}
}

func sortValue(img loadedImage, sortField SortField) string {
	switch sortField {
	case SortCaptured:
		return img.indexEntry.CapturedAt
	case SortProcessed:
		return img.indexEntry.ProcessedAt
	default:
		return ""
	}
}

func newTemplateData(images []loadedImage, urlPrefix string) (templateData, error) {
	data := templateData{
		Images: make([]templateImage, 0, len(images)),
	}

	for _, img := range images {
		tmplImg, err := newTemplateImage(img, urlPrefix)
		if err != nil {
			return templateData{}, fmt.Errorf("image %q: %w", img.indexEntry.Dir, err)
		}
		data.Images = append(data.Images, tmplImg)
	}

	return data, nil
}

func newTemplateImage(img loadedImage, urlPrefix string) (templateImage, error) {
	jpegVariants := sortedVariants(img.manifest.Variants[image.FormatJPEG])
	if len(jpegVariants) == 0 {
		return templateImage{}, errors.New("missing JPEG variants")
	}

	var sources []templateSource
	avifVariants := sortedVariants(img.manifest.Variants[image.FormatAVIF])
	if len(avifVariants) != 0 {
		sources = append(sources, templateSource{
			Type:   "image/avif",
			Srcset: srcset(img.indexEntry.Dir, avifVariants, urlPrefix),
			Sizes:  imageSizes,
		})
	}

	fallback := fallbackVariant(jpegVariants)
	return templateImage{
		Sources: sources,
		Fallback: templateFallback{
			Src:    publicURL(urlPrefix, img.indexEntry.Dir, fallback.Path),
			Srcset: srcset(img.indexEntry.Dir, jpegVariants, urlPrefix),
			Sizes:  imageSizes,
			Width:  fallback.Width,
			Height: fallback.Height,
			Alt:    altText(img.manifest),
		},
	}, nil
}

func sortedVariants(variants []manifest.VariantFile) []manifest.VariantFile {
	sorted := slices.Clone(variants)
	slices.SortFunc(sorted, func(a, b manifest.VariantFile) int {
		if a.Width != b.Width {
			return cmp.Compare(a.Width, b.Width)
		}
		return cmp.Compare(a.Path.String(), b.Path.String())
	})
	return sorted
}

func srcset(imgDir paths.RelPath, variants []manifest.VariantFile, urlPrefix string) string {
	items := make([]string, 0, len(variants))
	for _, variant := range variants {
		items = append(items, fmt.Sprintf("%s %dw", publicURL(urlPrefix, imgDir, variant.Path), variant.Width))
	}
	return strings.Join(items, ", ")
}

func fallbackVariant(variants []manifest.VariantFile) manifest.VariantFile {
	for _, variant := range variants {
		if variant.Width >= fallbackDisplayWidth {
			return variant
		}
	}
	return variants[len(variants)-1]
}

func altText(m manifest.Manifest) string {
	if m.Description != "" {
		return m.Description
	}
	return m.Title
}

func publicURL(urlPrefix string, imgDir paths.RelPath, file paths.RelPath) string {
	return path.Join(urlPrefix, imgDir.String(), file.String())
}

func loadImages(opts Options) ([]loadedImage, error) {
	idx, err := index.ReadDir(opts.ImagesRoot)
	if err != nil {
		return nil, fmt.Errorf("reading gallery index: %w", err)
	}

	images := make([]loadedImage, 0, len(idx.Images))
	for _, indexEntry := range idx.Images {
		manifestPath, err := paths.JoinAbs(opts.ImagesRoot, indexEntry.Manifest)
		if err != nil {
			return nil, fmt.Errorf("image %q: building manifest path %q: %w", indexEntry.Dir, indexEntry.Manifest, err)
		}

		mani, err := manifest.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("image %q: reading manifest %q: %w", indexEntry.Dir, indexEntry.Manifest, err)
		}

		images = append(images, loadedImage{
			indexEntry: indexEntry,
			manifest:   mani,
		})
	}

	return images, nil
}
