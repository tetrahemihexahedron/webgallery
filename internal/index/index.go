package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const filename = "index.json"

// Index describes the processed images in a directory.
type Index struct {
	GeneratedAt string  `json:"generatedAt"`
	Images      []Image `json:"images"`
}

// Image is one processed image entry in an Index.
type Image struct {
	Dir         string `json:"dir"`
	Manifest    string `json:"manifest"`
	Title       string `json:"title"`
	CapturedAt  string `json:"capturedAt"`
	ProcessedAt string `json:"processedAt"`
	SHA256      string `json:"sha256"`
}

// ImagesBySHA256 returns the index's images keyed by their SHA-256 hashes.
func (idx Index) ImagesBySHA256() (map[string]Image, error) {
	images := make(map[string]Image, len(idx.Images))

	for _, img := range idx.Images {
		if previous, ok := images[img.SHA256]; ok {
			return nil, fmt.Errorf("index contains duplicate sha256 %q for dirs %q and %q", img.SHA256, previous.Dir, img.Dir)
		}

		images[img.SHA256] = img
	}
	return images, nil
}

// Read reads index.json from dir. If the file does not exist, Read
// returns an empty Index.
func Read(dir string) (Index, error) {
	path := filepath.Join(dir, filename)
	var idx Index

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			idx.Images = []Image{}
			return idx, nil
		}
		return Index{}, fmt.Errorf("reading index %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, fmt.Errorf("parsing index %q: %w", path, err)
	}

	if idx.Images == nil {
		idx.Images = []Image{}
	}
	return idx, nil
}
