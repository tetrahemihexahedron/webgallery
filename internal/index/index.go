package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/manifest"
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

// IndexPath returns the absolute path to the collection index file in outRoot.
func IndexPath(outRoot paths.AbsPath) (paths.AbsPath, error) {
	idxRelPath, err := paths.NewRelPath(idxFilename)
	if err != nil {
		return paths.AbsPath{}, fmt.Errorf("building index path: %w", err)
	}

	path, err := paths.JoinAbs(outRoot, idxRelPath)
	if err != nil {
		return paths.AbsPath{}, fmt.Errorf("building index path: %w", err)
	}

	return path, nil
}

func imageFromProcessed(outRoot paths.AbsPath, img image.Processed) (Image, error) {
	if img.DirRelPath.String() == "" {
		return Image{}, fmt.Errorf("dir relative path is required")
	}

	imgDirAbsPath, err := paths.JoinAbs(outRoot, img.DirRelPath)
	if err != nil {
		return Image{}, fmt.Errorf("building image directory path: %w", err)
	}

	manifestAbsPath, err := manifest.ManifestPath(imgDirAbsPath)
	if err != nil {
		return Image{}, fmt.Errorf("building manifest path: %w", err)
	}

	manifestRelPath, err := relPathFromAbs(outRoot, manifestAbsPath)
	if err != nil {
		return Image{}, fmt.Errorf("building manifest relative path: %w", err)
	}

	return Image{
		Dir:         img.DirRelPath,
		Manifest:    manifestRelPath,
		Title:       img.Title,
		CapturedAt:  img.CapturedAt,
		ProcessedAt: img.ProcessedAt,
		SHA256:      img.Source.Hash,
	}, nil
}

func relPathFromAbs(base paths.AbsPath, target paths.AbsPath) (paths.RelPath, error) {
	relPath, err := filepath.Rel(base.String(), target.String())
	if err != nil {
		return paths.RelPath{}, err
	}

	return paths.NewRelPath(relPath)
}

// ImageDirsBySHA256 returns the index's image directories keyed by their SHA-256 hashes.
func (idx Index) ImageDirsBySHA256() (map[string]paths.RelPath, error) {
	imageDirs := make(map[string]paths.RelPath, len(idx.Images))

	for _, img := range idx.Images {
		if previousDir, ok := imageDirs[img.SHA256]; ok {
			return nil, fmt.Errorf("index contains duplicate sha256 %q for dirs %q and %q", img.SHA256, previousDir, img.Dir)
		}

		imageDirs[img.SHA256] = img.Dir
	}
	return imageDirs, nil
}

// Read reads index.json from dir. If the file does not exist, Read
// returns an empty Index.
func Read(dir paths.AbsPath) (Index, error) {
	path, err := IndexPath(dir)
	if err != nil {
		return Index{}, err
	}
	var file indexFile

	data, err := os.ReadFile(path.String())
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
