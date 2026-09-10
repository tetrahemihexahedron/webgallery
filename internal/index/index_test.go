package index_test

// Note: These tests assume Unix-style paths and are not Windows compatible.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/paths"
)

func TestReadReturnsEmptyIndexWhenFileIsMissing(t *testing.T) {
	dir := mustAbs(t, t.TempDir())

	got, err := index.Read(dir)
	if err != nil {
		t.Fatalf("index.Read(%q) returned error: %v", dir, err)
	}

	want := index.Index{Images: []index.Image{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("index.Read(%q) returned %+v, want %+v", dir, got, want)
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name string
		dir  paths.AbsPath
		want index.Index
	}{
		{
			name: "with images",
			dir:  mustAbs(t, filepath.Join("testdata", "valid")),
			want: index.Index{
				GeneratedAt: "2026-08-24T18:00:00Z",
				Images: []index.Image{
					{
						Dir:         mustRel(t, "2024/05/abc123"),
						Manifest:    mustRel(t, "2024/05/abc123/manifest.json"),
						Title:       "Rosie posing",
						CapturedAt:  "2024-05-12T14:22:00",
						ProcessedAt: "2026-08-24T18:00:00Z",
						SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
					},
				},
			},
		},
		{
			name: "with no images",
			dir:  mustAbs(t, filepath.Join("testdata", "no_images")),
			want: index.Index{
				GeneratedAt: "2026-08-24T18:00:00Z",
				Images:      []index.Image{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := index.Read(tc.dir)
			if err != nil {
				t.Fatalf("index.Read(%q) returned error: %v", tc.dir, err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("index.Read(%q) returned %+v, want %+v", tc.dir, got, tc.want)
			}
		})
	}
}

func TestReadReturnsErrorForMalformedJSON(t *testing.T) {
	dir := mustAbs(t, filepath.Join("testdata", "malformed"))

	_, err := index.Read(dir)
	if err == nil {
		t.Fatalf("index.Read(%q) returned nil error, want error", dir)
	}
	if !strings.Contains(err.Error(), "parsing index") {
		t.Errorf("index.Read(%q) error = %q, want message containing %q", dir, err, "parsing index")
	}
}

func TestReadReturnsErrorForUnreadableIndex(t *testing.T) {
	dirPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(dirPath, "index.json"), 0755); err != nil {
		t.Fatalf("making test directory: %v", err)
	}
	dir := mustAbs(t, dirPath)

	_, err := index.Read(dir)
	if err == nil {
		t.Fatalf("index.Read(%q) returned nil error, want error", dir)
	}
	if !strings.Contains(err.Error(), "reading index") {
		t.Errorf("index.Read(%q) error = %q, want message containing %q", dir, err, "reading index")
	}
}

func TestUpdateFile(t *testing.T) {
	existingImage := index.Image{
		Dir:         mustRel(t, "2024/05/abc123"),
		Manifest:    mustRel(t, "2024/05/abc123/manifest.json"),
		Title:       "Rosie posing",
		CapturedAt:  "2024-05-12T14:22:00",
		ProcessedAt: "2026-08-24T18:00:00Z",
		SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
	}
	newImage := image.Processed{
		Source: image.Source{
			Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		DirRelPath:  mustRel(t, "2025/01/def456"),
		Title:       "Rosie running",
		CapturedAt:  "2025-01-02T03:04:05",
		ProcessedAt: "2026-08-25T12:00:00Z",
	}

	tests := []struct {
		name                     string
		newImages                []image.Processed
		want                     index.Index
		wantGeneratedAtRefreshed bool
	}{
		{
			name:      "appends image",
			newImages: []image.Processed{newImage},
			want: index.Index{
				GeneratedAt: "<checked separately>",
				Images: []index.Image{
					existingImage,
					{
						Dir:         mustRel(t, "2025/01/def456"),
						Manifest:    mustRel(t, "2025/01/def456/manifest.json"),
						Title:       "Rosie running",
						CapturedAt:  "2025-01-02T03:04:05",
						ProcessedAt: "2026-08-25T12:00:00Z",
						SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				},
			},
			wantGeneratedAtRefreshed: true,
		},
		{
			name:      "with no images",
			newImages: []image.Processed{},
			want: index.Index{
				GeneratedAt: "2026-08-24T18:00:00Z",
				Images:      []index.Image{existingImage},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outRoot := mustAbs(t, t.TempDir())
			copyValidIndexFixture(t, outRoot)

			idx, err := index.Read(outRoot)
			if err != nil {
				t.Fatalf("index.Read(%q) returned error before update: %v", outRoot, err)
			}

			before := time.Now().UTC().Add(-1 * time.Second)
			if err := idx.UpdateFile(outRoot, tc.newImages); err != nil {
				t.Fatalf("Index.UpdateFile(%q) returned error: %v", outRoot, err)
			}
			after := time.Now().UTC().Add(time.Second)

			got, err := index.Read(outRoot)
			if err != nil {
				t.Fatalf("index.Read(%q) returned error after update: %v", outRoot, err)
			}

			if !reflect.DeepEqual(idx, got) {
				t.Errorf("receiver index = %+v, want written index %+v", idx, got)
			}

			if tc.wantGeneratedAtRefreshed {
				generatedAt, err := time.Parse(time.RFC3339, got.GeneratedAt)
				if err != nil {
					t.Fatalf("updated index GeneratedAt = %q, want RFC3339 time", got.GeneratedAt)
				}
				if generatedAt.Before(before) || generatedAt.After(after) {
					t.Errorf("updated index GeneratedAt = %v, want between %v and %v", generatedAt, before, after)
				}
				got.GeneratedAt = tc.want.GeneratedAt
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("updated index = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestImageDirsBySHA256(t *testing.T) {
	imgA := index.Image{
		Dir:    mustRel(t, "2024/05/abc123"),
		SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	imgB := index.Image{
		Dir:    mustRel(t, "2024/05/def456"),
		SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}

	tests := []struct {
		name string
		idx  index.Index
		want map[string]paths.RelPath
	}{
		{
			name: "empty index",
			idx:  index.Index{},
			want: map[string]paths.RelPath{},
		},
		{
			name: "no images",
			idx:  index.Index{Images: []index.Image{}},
			want: map[string]paths.RelPath{},
		},
		{
			name: "multiple images",
			idx:  index.Index{Images: []index.Image{imgA, imgB}},
			want: map[string]paths.RelPath{
				imgA.SHA256: imgA.Dir,
				imgB.SHA256: imgB.Dir,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.idx.ImageDirsBySHA256()
			if err != nil {
				t.Fatalf("Index.ImageDirsBySHA256() returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Index.ImageDirsBySHA256() returned %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestImageDirsBySHA256ReturnsErrorForDuplicateHash(t *testing.T) {
	idx := index.Index{
		Images: []index.Image{
			{
				Dir:    mustRel(t, "2024/05/abc123"),
				SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				Dir:    mustRel(t, "2024/05/def456"),
				SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}

	_, err := idx.ImageDirsBySHA256()
	if err == nil {
		t.Fatalf("Index.ImageDirsBySHA256() returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "duplicate sha256") {
		t.Errorf("Index.ImageDirsBySHA256() error = %q, want message containing %q", err, "duplicate sha256")
	}
}

func copyValidIndexFixture(t *testing.T, destDir paths.AbsPath) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "valid", "index.json"))
	if err != nil {
		t.Fatalf("reading valid index fixture: %v", err)
	}

	destPath, err := index.IndexPath(destDir)
	if err != nil {
		t.Fatalf("building destination index path: %v", err)
	}
	if err := os.WriteFile(destPath.String(), data, 0644); err != nil {
		t.Fatalf("writing valid index fixture to %q: %v", destPath, err)
	}
}

func mustAbs(t *testing.T, path string) paths.AbsPath {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v, want nil", path, err)
	}

	p, err := paths.NewAbsPath(absPath)
	if err != nil {
		t.Fatalf("paths.NewAbsPath(%q) error = %v, want nil", absPath, err)
	}

	return p
}

func mustRel(t *testing.T, path string) paths.RelPath {
	t.Helper()

	p, err := paths.NewRelPath(path)
	if err != nil {
		t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", path, err)
	}

	return p
}
