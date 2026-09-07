package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webimage/internal/paths"
)

const idxFilename = "index.json"

// Index describes the processed images in a directory.
type Index struct {
	GeneratedAt string
	Images      []Image
}

// Image is one processed image entry in an Index.
type Image struct {
	Dir         paths.RelPath
	Manifest    paths.RelPath
	Title       string
	CapturedAt  string
	ProcessedAt string
	SHA256      string
}

type indexFile struct {
	GeneratedAt string      `json:"generatedAt"`
	Images      []imageFile `json:"images"`
}

type imageFile struct {
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
func Read(dir paths.AbsPath) (Index, error) {
	path := filepath.Join(dir.String(), idxFilename)
	var file indexFile

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Index{Images: []Image{}}, nil
		}
		return Index{}, fmt.Errorf("reading index %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &file); err != nil {
		return Index{}, fmt.Errorf("parsing index %q: %w", path, err)
	}

	idx, err := parseIndex(file)
	if err != nil {
		return Index{}, fmt.Errorf("parsing index %q: %w", path, err)
	}

	return idx, nil
}

func parseIndex(file indexFile) (Index, error) {
	idx := Index{
		GeneratedAt: file.GeneratedAt,
		Images:      make([]Image, 0, len(file.Images)),
	}

	for i, imgFile := range file.Images {
		img, err := parseImage(imgFile)
		if err != nil {
			return Index{}, fmt.Errorf("image %d: %w", i, err)
		}
		idx.Images = append(idx.Images, img)
	}

	return idx, nil
}

func parseImage(file imageFile) (Image, error) {
	dir, err := paths.NewRelPath(file.Dir)
	if err != nil {
		return Image{}, fmt.Errorf("dir: %w", err)
	}

	manifest, err := paths.NewRelPath(file.Manifest)
	if err != nil {
		return Image{}, fmt.Errorf("manifest: %w", err)
	}

	return Image{
		Dir:         dir,
		Manifest:    manifest,
		Title:       file.Title,
		CapturedAt:  file.CapturedAt,
		ProcessedAt: file.ProcessedAt,
		SHA256:      file.SHA256,
	}, nil
}
