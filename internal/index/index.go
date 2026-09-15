package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

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

// UpdateFile appends newImages to idx and writes the updated index file in outRoot.
func (idx *Index) UpdateFile(outRoot paths.AbsPath, newImages []image.Processed) error {
	if len(newImages) == 0 {
		return nil
	}

	return idx.updateFileAt(outRoot, newImages, time.Now())
}

func (idx *Index) updateFileAt(outRoot paths.AbsPath, newImages []image.Processed, generatedAt time.Time) error {
	updated, err := updatedIndex(*idx, outRoot, newImages, generatedAt)
	if err != nil {
		return err
	}

	if err := writeIndexFile(outRoot, updated); err != nil {
		return err
	}

	*idx = updated
	return nil
}

func updatedIndex(idx Index, outRoot paths.AbsPath, newImages []image.Processed, generatedAt time.Time) (Index, error) {
	updated := Index{
		GeneratedAt: formatGeneratedAt(generatedAt),
		Images:      make([]Image, 0, len(idx.Images)+len(newImages)),
	}
	updated.Images = append(updated.Images, idx.Images...)

	for i, img := range newImages {
		indexImg, err := imageFromProcessed(outRoot, img)
		if err != nil {
			return Index{}, fmt.Errorf("new image %d: %w", i, err)
		}

		updated.Images = append(updated.Images, indexImg)
	}

	return updated, nil
}

func writeIndexFile(outRoot paths.AbsPath, idx Index) error {
	path, err := IndexPath(outRoot)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(indexToFile(idx), "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	tmpFile, err := os.CreateTemp(outRoot.String(), ".index-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary index file in %q: %w", outRoot, err)
	}
	tmpPath := tmpFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(jsonBytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing temporary index %q: %w", tmpPath, err)
	}
	if err := tmpFile.Chmod(0644); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("setting temporary index permissions %q: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temporary index %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path.String()); err != nil {
		return fmt.Errorf("renaming temporary index %q to %q: %w", tmpPath, path, err)
	}
	removeTemp = false

	return nil
}

func formatGeneratedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func indexToFile(idx Index) indexFile {
	file := indexFile{
		GeneratedAt: idx.GeneratedAt,
		Images:      make([]imageFile, 0, len(idx.Images)),
	}

	for _, img := range idx.Images {
		file.Images = append(file.Images, imageFile{
			Dir:         img.Dir.String(),
			Manifest:    img.Manifest.String(),
			Title:       img.Title,
			CapturedAt:  img.CapturedAt,
			ProcessedAt: img.ProcessedAt,
			SHA256:      img.SHA256,
		})
	}

	return file
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

// ReadDir reads index.json from dir. If the file does not exist, ReadDir
// returns an empty Index.
func ReadDir(dir paths.AbsPath) (Index, error) {
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
