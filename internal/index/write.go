package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/paths"
)

// Update appends newImages, writes the resulting index.json in outRoot,
// and updates idx only after the write succeeds.
func (idx *Index) Update(outRoot paths.AbsPath, newImages []image.Processed) error {
	if len(newImages) == 0 {
		return nil
	}

	return idx.updateAt(outRoot, newImages, time.Now())
}

func (idx *Index) updateAt(outRoot paths.AbsPath, newImages []image.Processed, generatedAt time.Time) error {
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
		Images:      make([]Entry, 0, len(idx.Images)+len(newImages)),
	}
	updated.Images = append(updated.Images, idx.Images...)

	for i, img := range newImages {
		entry, err := entryFromProcessed(outRoot, img)
		if err != nil {
			return Index{}, fmt.Errorf("new image %d: %w", i, err)
		}

		updated.Images = append(updated.Images, entry)
	}

	return updated, nil
}

func writeIndexFile(outRoot paths.AbsPath, idx Index) error {
	path, err := IndexPath(outRoot)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(indexToJSON(idx), "", "  ")
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

func indexToJSON(idx Index) indexJSON {
	file := indexJSON{
		GeneratedAt: idx.GeneratedAt,
		Images:      make([]entryJSON, 0, len(idx.Images)),
	}

	for _, entry := range idx.Images {
		file.Images = append(file.Images, entryJSON{
			Dir:         entry.Dir.String(),
			Manifest:    entry.Manifest.String(),
			Title:       entry.Title,
			CapturedAt:  entry.CapturedAt,
			ProcessedAt: entry.ProcessedAt,
			SHA256:      entry.SHA256,
		})
	}

	return file
}

func entryFromProcessed(outRoot paths.AbsPath, img image.Processed) (Entry, error) {
	if img.DirRelPath.String() == "" {
		return Entry{}, fmt.Errorf("dir relative path is required")
	}

	imgDirAbsPath, err := paths.JoinAbs(outRoot, img.DirRelPath)
	if err != nil {
		return Entry{}, fmt.Errorf("building image directory path: %w", err)
	}

	manifestAbsPath, err := manifest.ManifestPath(imgDirAbsPath)
	if err != nil {
		return Entry{}, fmt.Errorf("building manifest path: %w", err)
	}

	manifestRelPath, err := relPathFromAbs(outRoot, manifestAbsPath)
	if err != nil {
		return Entry{}, fmt.Errorf("building manifest relative path: %w", err)
	}

	return Entry{
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
