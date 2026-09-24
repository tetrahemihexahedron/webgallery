package index

import (
	"fmt"

	"tetrahemihexahedron/webgallery/internal/paths"
)

const idxFilename = "index.json"

// Index describes the processed images in a directory.
type Index struct {
	GeneratedAt string
	Images      []Entry
}

// Entry is one processed image entry in an Index.
type Entry struct {
	Dir         paths.RelPath
	Manifest    paths.RelPath
	Title       string
	CapturedAt  string
	ProcessedAt string
	SHA256      string
}

type indexJSON struct {
	GeneratedAt string      `json:"generatedAt"`
	Images      []entryJSON `json:"images"`
}

type entryJSON struct {
	Dir         string `json:"dir"`
	Manifest    string `json:"manifest"`
	Title       string `json:"title"`
	CapturedAt  string `json:"capturedAt"`
	ProcessedAt string `json:"processedAt"`
	SHA256      string `json:"sha256"`
}

func indexPath(outRoot paths.AbsPath) (paths.AbsPath, error) {
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

// ImageDirsBySHA256 returns the index's image directories keyed by their SHA-256 hashes.
func (idx Index) ImageDirsBySHA256() (map[string]paths.RelPath, error) {
	imageDirs := make(map[string]paths.RelPath, len(idx.Images))

	for _, entry := range idx.Images {
		if previousDir, ok := imageDirs[entry.SHA256]; ok {
			return nil, fmt.Errorf("index contains duplicate sha256 %q for dirs %q and %q", entry.SHA256, previousDir, entry.Dir)
		}

		imageDirs[entry.SHA256] = entry.Dir
	}
	return imageDirs, nil
}
