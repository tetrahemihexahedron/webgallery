package index

import (
	"encoding/json"
	"fmt"
	"os"

	"tetrahemihexahedron/webgallery/internal/paths"
)

// ReadDir reads index.json from dir.
func ReadDir(dir paths.AbsPath) (Index, error) {
	path, err := indexPath(dir)
	if err != nil {
		return Index{}, err
	}
	var file indexJSON

	data, err := os.ReadFile(path.String())
	if err != nil {
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

func parseIndex(file indexJSON) (Index, error) {
	idx := Index{
		GeneratedAt: file.GeneratedAt,
		Images:      make([]Entry, 0, len(file.Images)),
	}

	for i, entryFile := range file.Images {
		entry, err := parseEntry(entryFile)
		if err != nil {
			return Index{}, fmt.Errorf("image %d: %w", i, err)
		}
		idx.Images = append(idx.Images, entry)
	}

	return idx, nil
}

func parseEntry(file entryJSON) (Entry, error) {
	dir, err := paths.NewRelPath(file.Dir)
	if err != nil {
		return Entry{}, fmt.Errorf("dir: %w", err)
	}

	manifest, err := paths.NewRelPath(file.Manifest)
	if err != nil {
		return Entry{}, fmt.Errorf("manifest: %w", err)
	}

	return Entry{
		Dir:         dir,
		Manifest:    manifest,
		Title:       file.Title,
		CapturedAt:  file.CapturedAt,
		ProcessedAt: file.ProcessedAt,
		SHA256:      file.SHA256,
	}, nil
}
