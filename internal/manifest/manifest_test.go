package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/paths"
)

type manifestJSON struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	CapturedAt  string                   `json:"capturedAt"`
	ProcessedAt string                   `json:"processedAt"`
	SHA256      string                   `json:"sha256"`
	Width       int                      `json:"width"`
	Height      int                      `json:"height"`
	Variants    map[string][]variantJSON `json:"variants"`
}

type variantJSON struct {
	Path   string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func TestReadFile(t *testing.T) {
	got, err := manifest.ReadFile(testdataPath(t, "valid.json"))
	if err != nil {
		t.Fatalf("manifest.ReadFile() returned error: %v", err)
	}

	want := manifest.Manifest{
		Title:       "2023 October Posing",
		Description: "Rosie as a small puppy, sitting and looking directly at the camera.",
		CapturedAt:  "2023-10-03T17:26:39",
		ProcessedAt: "2026-08-24T18:00:00Z",
		SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
		Width:       800,
		Height:      1067,
		Variants: map[image.Format][]manifest.VariantFile{
			image.FormatJPEG: {
				{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 534},
				{Path: mustRel(t, "w800.jpg"), Width: 800, Height: 1067},
			},
			image.FormatAVIF: {
				{Path: mustRel(t, "w400.avif"), Width: 400, Height: 534},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("manifest.ReadFile() mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestReadFileReturnsError(t *testing.T) {
	tests := []struct {
		name            string
		fixtureFilename string
		wantMessage     string
	}{
		{
			name:            "malformed JSON",
			fixtureFilename: "malformed.json",
			wantMessage:     "parsing manifest",
		},
		{
			name:            "invalid variant path",
			fixtureFilename: "invalid_variant_path.json",
			wantMessage:     `variant "JPEG" file 0 src`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manifest.ReadFile(testdataPath(t, tc.fixtureFilename))
			if err == nil {
				t.Fatalf("manifest.ReadFile() returned nil error, want error")
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("manifest.ReadFile() error = %q, want containing %q", err, tc.wantMessage)
			}
		})
	}
}

func TestFromProcessed(t *testing.T) {
	img := image.Processed{
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
		},
	}
	want := manifest.Manifest{
		Title:       img.Title,
		Description: img.Description,
		CapturedAt:  img.CapturedAt,
		ProcessedAt: img.ProcessedAt,
		SHA256:      img.Source.Hash,
		Width:       img.Source.Width,
		Height:      img.Source.Height,
		Variants: map[image.Format][]manifest.VariantFile{
			image.FormatJPEG: {
				{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 534},
				{Path: mustRel(t, "w800.jpg"), Width: 800, Height: 1067},
			},
		},
	}

	got, err := manifest.FromProcessed(img)
	if err != nil {
		t.Fatalf("manifest.FromProcessed() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("manifest.FromProcessed() mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestFromProcessedReturnsErrorForUnsupportedFormat(t *testing.T) {
	img := image.Processed{
		Variants: []image.Variant{{Format: image.FormatOther}},
	}

	_, err := manifest.FromProcessed(img)
	if err == nil {
		t.Fatal("manifest.FromProcessed() returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("manifest.FromProcessed() error = %q, want message containing %q", err, "unsupported format")
	}
}

func TestWriteFile(t *testing.T) {
	tests := []struct {
		name string
		mani manifest.Manifest
		want manifestJSON
	}{
		{
			name: "writes complete metadata",
			mani: manifest.Manifest{
				Title:       "2023 October Posing",
				Description: "Rosie as a small puppy, sitting and looking directly at the camera.",
				CapturedAt:  "2023-10-03T17:26:39",
				ProcessedAt: "2026-08-24T18:00:00Z",
				SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
				Width:       800,
				Height:      1067,
				Variants: map[image.Format][]manifest.VariantFile{
					image.FormatJPEG: {
						{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 534},
						{Path: mustRel(t, "w800.jpg"), Width: 800, Height: 1067},
					},
					image.FormatAVIF: {
						{Path: mustRel(t, "w400.avif"), Width: 400, Height: 534},
					},
				},
			},
			want: manifestJSON{
				Title:       "2023 October Posing",
				Description: "Rosie as a small puppy, sitting and looking directly at the camera.",
				CapturedAt:  "2023-10-03T17:26:39",
				ProcessedAt: "2026-08-24T18:00:00Z",
				SHA256:      "7f43b6f0a877e8590c4f0c7d55d99b188a671de7bf58156ac0d3ac38df842cc9",
				Width:       800,
				Height:      1067,
				Variants: map[string][]variantJSON{
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
			mani: manifest.Manifest{
				Width:    4032,
				Height:   3024,
				Variants: map[image.Format][]manifest.VariantFile{},
			},
			want: manifestJSON{
				Width:    4032,
				Height:   3024,
				Variants: map[string][]variantJSON{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := mustAbs(t, t.TempDir())
			path, err := manifest.ManifestPath(dir)
			if err != nil {
				t.Fatalf("manifest.ManifestPath(%q) returned error: %v", dir, err)
			}

			if err := manifest.WriteFile(path, tc.mani); err != nil {
				t.Fatalf("manifest.WriteFile(%q, %+v) returned error: %v", path, tc.mani, err)
			}

			got := readManifest(t, dir)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("manifest.WriteFile(%q, %+v) manifest mismatch\n got: %+v\nwant: %+v", path, tc.mani, got, tc.want)
			}
		})
	}
}

func TestWriteFileReturnsErrorForMissingDir(t *testing.T) {
	path := mustAbs(t, filepath.Join(t.TempDir(), "missing", "manifest.json"))

	if err := manifest.WriteFile(path, manifest.Manifest{}); err == nil {
		t.Fatalf("manifest.WriteFile(%q, manifest.Manifest{}) returned nil error, want error", path)
	}
}

func TestWriteFileReturnsErrorForUnsupportedFormat(t *testing.T) {
	path := mustAbs(t, filepath.Join(t.TempDir(), "manifest.json"))
	mani := manifest.Manifest{
		Variants: map[image.Format][]manifest.VariantFile{
			image.FormatOther: {},
		},
	}

	err := manifest.WriteFile(path, mani)
	if err == nil {
		t.Fatalf("manifest.WriteFile(%q, %+v) returned nil error, want error", path, mani)
	}
	if !strings.Contains(err.Error(), "unsupported variant format") {
		t.Errorf("manifest.WriteFile() error = %q, want message containing %q", err, "unsupported variant format")
	}
}

func readManifest(t *testing.T, dir paths.AbsPath) manifestJSON {
	t.Helper()

	path := filepath.Join(dir.String(), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading manifest %q: %v", path, err)
	}

	var got manifestJSON
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling manifest %q: %v", path, err)
	}

	return got
}

func testdataPath(t *testing.T, name string) paths.AbsPath {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v, want nil", name, err)
	}

	return mustAbs(t, path)
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
