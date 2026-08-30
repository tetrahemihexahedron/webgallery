package manifest

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webimage/internal/image"
)

type manifest struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	CapturedAt  string                 `json:"capturedAt"`
	ProcessedAt string                 `json:"processedAt"`
	SHA256      string                 `json:"sha256"`
	Width       int                    `json:"width"`
	Height      int                    `json:"height"`
	Variants    map[string][]imageFile `json:"variants"`
}

type imageFile struct {
	Path   string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func Write(img image.Processed) error {
	mani := manifest{
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

	outPath := filepath.Join(img.DirAbsPath.String(), "manifest.json")
	if err = os.WriteFile(outPath, jsonBytes, 0644); err != nil {
		return err
	}

	return nil
}

func manifestVariants(i image.Processed) map[string][]imageFile {
	maniVariants := make(map[string][]imageFile)

	aspectRatio := float64(i.Source.Height) / float64(i.Source.Width)
	for _, v := range i.Variants {
		format := v.Format.String()
		maniVariants[format] = append(
			maniVariants[format],
			imageFile{
				Path:   v.Path,
				Width:  v.Width,
				Height: int(math.Round(float64(v.Width) * aspectRatio)),
			},
		)
	}
	return maniVariants
}
