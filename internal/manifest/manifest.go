package manifest

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/paths"
)

const filename = "manifest.json"

// ManifestPath returns the absolute path to an image directory's manifest file.
func ManifestPath(dir paths.AbsPath) (paths.AbsPath, error) {
	manifestRelPath, err := paths.NewRelPath(filename)
	if err != nil {
		return paths.AbsPath{}, err
	}

	return paths.JoinAbs(dir, manifestRelPath)
}

// Manifest describes a processed image manifest.
type Manifest struct {
	Title       string
	Description string
	CapturedAt  string
	ProcessedAt string
	SHA256      string
	Width       int
	Height      int
	Variants    map[image.Format][]VariantFile
}

// VariantFile describes one generated image file in a manifest.
type VariantFile struct {
	Path   paths.RelPath
	Width  int
	Height int
}

type manifestJSON struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	CapturedAt  string                   `json:"capturedAt"`
	ProcessedAt string                   `json:"processedAt"`
	SHA256      string                   `json:"sha256"`
	Width       int                      `json:"width"`
	Height      int                      `json:"height"`
	Variants    map[string][]variantJSON `json:"variants"`
}

type variantJSON struct {
	Path   string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// FromProcessed creates a manifest from a processed image.
func FromProcessed(img image.Processed) (Manifest, error) {
	if err := validateVariantFormats(img.Variants); err != nil {
		return Manifest{}, err
	}

	return Manifest{
		Title:       img.Title,
		Description: img.Description,
		CapturedAt:  img.CapturedAt,
		ProcessedAt: img.ProcessedAt,
		SHA256:      img.Source.Hash,
		Width:       img.Source.Width,
		Height:      img.Source.Height,
		Variants:    variantsFromProcessed(img.Variants),
	}, nil
}

// WriteFile writes mani as JSON to path.
func WriteFile(path paths.AbsPath, mani Manifest) error {
	if err := validateManifestVariantFormats(mani.Variants); err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(manifestToJSON(mani), "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path.String(), jsonBytes, 0644)
}

// ReadFile reads a processed image manifest from path.
func ReadFile(path paths.AbsPath) (Manifest, error) {
	data, err := os.ReadFile(path.String())
	if err != nil {
		return Manifest{}, fmt.Errorf("reading manifest %q: %w", path, err)
	}

	var file manifestJSON
	if err := json.Unmarshal(data, &file); err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest %q: %w", path, err)
	}

	mani, err := manifestFromJSON(file)
	if err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest %q: %w", path, err)
	}

	return mani, nil
}

func variantsFromProcessed(processed []image.Variant) map[image.Format][]VariantFile {
	variants := make(map[image.Format][]VariantFile)

	for _, variant := range processed {
		variants[variant.Format] = append(
			variants[variant.Format],
			VariantFile{
				Path:   variant.Path,
				Width:  variant.Width,
				Height: variant.Height,
			},
		)
	}
	return variants
}

func manifestToJSON(mani Manifest) manifestJSON {
	file := manifestJSON{
		Title:       mani.Title,
		Description: mani.Description,
		CapturedAt:  mani.CapturedAt,
		ProcessedAt: mani.ProcessedAt,
		SHA256:      mani.SHA256,
		Width:       mani.Width,
		Height:      mani.Height,
		Variants:    make(map[string][]variantJSON, len(mani.Variants)),
	}

	for format, variants := range mani.Variants {
		sorted := slices.Clone(variants)
		slices.SortFunc(sorted, func(a, b VariantFile) int {
			if a.Width != b.Width {
				return cmp.Compare(a.Width, b.Width)
			}
			return cmp.Compare(a.Path.String(), b.Path.String())
		})

		for _, variant := range sorted {
			file.Variants[format.String()] = append(file.Variants[format.String()], variantJSON{
				Path:   variant.Path.String(),
				Width:  variant.Width,
				Height: variant.Height,
			})
		}
	}

	return file
}

func isSupportedVariantFormat(f image.Format) bool {
	switch f {
	case image.FormatJPEG, image.FormatAVIF:
		return true
	default:
		return false
	}
}

func validateVariantFormats(variants []image.Variant) error {
	for i, variant := range variants {
		if !isSupportedVariantFormat(variant.Format) {
			return fmt.Errorf("variant %d has unsupported format %q", i, variant.Format)
		}
	}

	return nil
}

func validateManifestVariantFormats(variants map[image.Format][]VariantFile) error {
	for format := range variants {
		if !isSupportedVariantFormat(format) {
			return fmt.Errorf("unsupported variant format %q", format)
		}
	}

	return nil
}

func manifestFromJSON(file manifestJSON) (Manifest, error) {
	mani := Manifest{
		Title:       file.Title,
		Description: file.Description,
		CapturedAt:  file.CapturedAt,
		ProcessedAt: file.ProcessedAt,
		SHA256:      file.SHA256,
		Width:       file.Width,
		Height:      file.Height,
		Variants:    make(map[image.Format][]VariantFile, len(file.Variants)),
	}

	for formatName, files := range file.Variants {
		format := image.ParseFormat(formatName)
		if !isSupportedVariantFormat(format) {
			return Manifest{}, fmt.Errorf("unsupported variant format %q", formatName)
		}

		for i, file := range files {
			path, err := paths.NewRelPath(file.Path)
			if err != nil {
				return Manifest{}, fmt.Errorf("variant %q file %d src: %w", formatName, i, err)
			}

			mani.Variants[format] = append(mani.Variants[format], VariantFile{
				Path:   path,
				Width:  file.Width,
				Height: file.Height,
			})
		}
	}

	return mani, nil
}
