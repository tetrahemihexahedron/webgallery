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
