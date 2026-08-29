package index_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
						Dir:         "2024/05/abc123",
						Manifest:    "2024/05/abc123/manifest.json",
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

func TestImagesBySHA256(t *testing.T) {
	imgA := index.Image{
		Dir:    "2024/05/abc123",
		SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	imgB := index.Image{
		Dir:    "2024/05/def456",
		SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}

	tests := []struct {
		name string
		idx  index.Index
		want map[string]index.Image
	}{
		{
			name: "empty index",
			idx:  index.Index{},
			want: map[string]index.Image{},
		},
		{
			name: "no images",
			idx:  index.Index{Images: []index.Image{}},
			want: map[string]index.Image{},
		},
		{
			name: "multiple images",
			idx:  index.Index{Images: []index.Image{imgA, imgB}},
			want: map[string]index.Image{
				imgA.SHA256: imgA,
				imgB.SHA256: imgB,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.idx.ImagesBySHA256()
			if err != nil {
				t.Fatalf("Index.ImagesBySHA256() returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Index.ImagesBySHA256() returned %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestImagesBySHA256ReturnsErrorForDuplicateHash(t *testing.T) {
	idx := index.Index{
		Images: []index.Image{
			{
				Dir:    "2024/05/abc123",
				SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				Dir:    "2024/05/def456",
				SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}

	_, err := idx.ImagesBySHA256()
	if err == nil {
		t.Fatalf("Index.ImagesBySHA256() returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "duplicate sha256") {
		t.Errorf("Index.ImagesBySHA256() error = %q, want message containing %q", err, "duplicate sha256")
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
