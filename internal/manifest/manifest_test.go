package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/paths"
)

type manifestFile struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	CapturedAt  string                 `json:"capturedAt"`
	ProcessedAt string                 `json:"processedAt"`
	SHA256      string                 `json:"sha256"`
	Width       int                    `json:"width"`
	Height      int                    `json:"height"`
	Variants    map[string][]imageFile `json:"variants"`
}

type imageFile struct {
	Path   string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name string
		img  image.Processed
		want manifestFile
	}{
		{
			name: "writes complete metadata",
			img: image.Processed{
				Source: image.Source{
					Hash:   "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
					Width:  800,
					Height: 1067,
				},
				Title:       "2023 October Posing",
				Description: "Rosie as a small puppy, sitting and looking directly at the camera.",
				CapturedAt:  "2023-10-03T17:26:39",
				ProcessedAt: "2026-08-24T18:00:00Z",
				Variants: []image.Variant{
					{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400},
					{Path: mustRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800},
					{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400},
				},
			},
			want: manifestFile{
				Title:       "2023 October Posing",
				Description: "Rosie as a small puppy, sitting and looking directly at the camera.",
				CapturedAt:  "2023-10-03T17:26:39",
				ProcessedAt: "2026-08-24T18:00:00Z",
				SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
				Width:       800,
				Height:      1067,
				Variants: map[string][]imageFile{
					"JPEG": {
						{Path: "w400.jpg", Width: 400, Height: 534},
						{Path: "w800.jpg", Width: 800, Height: 1067},
					},
					"AVIF": {
						{Path: "w400.avif", Width: 400, Height: 534},
					},
				},
			},
		},
		{
			name: "handles no variants",
			img: image.Processed{
				Source: image.Source{
					Width:  4032,
					Height: 3024,
				},
			},
			want: manifestFile{
				Width:    4032,
				Height:   3024,
				Variants: map[string][]imageFile{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := tc.img
			img.DirAbsPath = mustAbs(t, t.TempDir())

			if err := manifest.Write(img); err != nil {
				t.Fatalf("manifest.Write(%+v) returned error: %v", img, err)
			}

			got := readManifest(t, img.DirAbsPath)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("manifest.Write(%+v) manifest mismatch\n got: %+v\nwant: %+v", img, got, tc.want)
			}
		})
	}
}

func TestWriteReturnsErrorForMissingDir(t *testing.T) {
	dir := mustAbs(t, filepath.Join(t.TempDir(), "missing"))
	img := image.Processed{DirAbsPath: dir}

	if err := manifest.Write(img); err == nil {
		t.Fatalf("manifest.Write(%+v) returned nil error, want error", img)
	}
}

func readManifest(t *testing.T, dir paths.AbsPath) manifestFile {
	t.Helper()

	path := filepath.Join(dir.String(), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading manifest %q: %v", path, err)
	}

	var got manifestFile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling manifest %q: %v", path, err)
	}

	return got
}

func mustAbs(t *testing.T, path string) paths.AbsPath {
	t.Helper()

	p, err := paths.NewAbsPath(path)
	if err != nil {
		t.Fatalf("paths.NewAbsPath(%q) error = %v, want nil", path, err)
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
