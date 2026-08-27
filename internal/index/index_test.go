package index_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/index"
)

func TestReadReturnsEmptyIndexWhenFileIsMissing(t *testing.T) {
	dir := t.TempDir()

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
		dir  string
		want index.Index
	}{
		{
			name: "with images",
			dir:  filepath.Join("testdata", "valid"),
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
			dir:  filepath.Join("testdata", "no_images"),
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
	dir := filepath.Join("testdata", "malformed")

	_, err := index.Read(dir)
	if err == nil {
		t.Fatalf("index.Read(%q) returned nil error, want error", dir)
	}
	if !strings.Contains(err.Error(), "parsing index") {
		t.Errorf("index.Read(%q) error = %q, want message containing %q", dir, err, "parsing index")
	}
}

func TestReadReturnsErrorForUnreadableIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "index.json"), 0755); err != nil {
		t.Fatalf("making test directory: %v", err)
	}

	_, err := index.Read(dir)
	if err == nil {
		t.Fatalf("index.Read(%q) returned nil error, want error", dir)
	}
	if !strings.Contains(err.Error(), "reading index") {
		t.Errorf("index.Read(%q) error = %q, want message containing %q", dir, err, "reading index")
	}
}
