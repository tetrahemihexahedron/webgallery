package manifest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

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
	Variants    map[image.Format][]File
}

// File describes one generated image file in a manifest.
type File struct {
	Path   paths.RelPath
	Width  int
	Height int
}

type manifestFile struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	CapturedAt  string                   `json:"capturedAt"`
	ProcessedAt string                   `json:"processedAt"`
	SHA256      string                   `json:"sha256"`
	Width       int                      `json:"width"`
	Height      int                      `json:"height"`
	Variants    map[string][]variantFile `json:"variants"`
}

type variantFile struct {
	Path   string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func Write(imgDir paths.AbsPath, img image.Processed) error {
	if err := validateVariantFormats(img.Variants); err != nil {
		return err
	}

	mani := manifestFile{
		Title:       img.Title,
		Description: img.Description,
		CapturedAt:  img.CapturedAt,
		ProcessedAt: img.ProcessedAt,
		SHA256:      img.Source.Hash,
		Width:       img.Source.Width,
		Height:      img.Source.Height,
		Variants:    manifestVariants(img),
	}

	jsonBytes, err := json.MarshalIndent(mani, "", "  ")
	if err != nil {
		return err
	}

	outPath, err := ManifestPath(imgDir)
	if err != nil {
		return err
	}
	if err = os.WriteFile(outPath.String(), jsonBytes, 0644); err != nil {
		return err
	}

	return nil
}

// ReadFile reads a processed image manifest from path.
func ReadFile(path paths.AbsPath) (Manifest, error) {
	data, err := os.ReadFile(path.String())
	if err != nil {
		return Manifest{}, fmt.Errorf("reading manifest %q: %w", path, err)
	}

	var file manifestFile
	if err := json.Unmarshal(data, &file); err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest %q: %w", path, err)
	}

	mani, err := manifestFromFile(file)
	if err != nil {
		return Manifest{}, fmt.Errorf("parsing manifest %q: %w", path, err)
	}

	return mani, nil
}

func manifestVariants(i image.Processed) map[string][]variantFile {
	maniVariants := make(map[string][]variantFile)

	aspectRatio := float64(i.Source.Height) / float64(i.Source.Width)
	for _, v := range i.Variants {
		format := v.Format.String()
		maniVariants[format] = append(
			maniVariants[format],
			variantFile{
				Path:   v.Path.String(),
				Width:  v.Width,
				Height: int(math.Round(float64(v.Width) * aspectRatio)),
			},
		)
	}
	return maniVariants
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

func manifestFromFile(file manifestFile) (Manifest, error) {
	mani := Manifest{
		Title:       file.Title,
		Description: file.Description,
		CapturedAt:  file.CapturedAt,
		ProcessedAt: file.ProcessedAt,
		SHA256:      file.SHA256,
		Width:       file.Width,
		Height:      file.Height,
		Variants:    make(map[image.Format][]File, len(file.Variants)),
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

			mani.Variants[format] = append(mani.Variants[format], File{
				Path:   path,
				Width:  file.Width,
				Height: file.Height,
			})
		}
	}

	return mani, nil
}
