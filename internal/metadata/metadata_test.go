package metadata_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/paths"
)

func TestExiftoolRead(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		wantMetadata []image.Metadata
	}{
		{
			name: "reads complete metadata",
			file: "complete_metadata.jpg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "complete_metadata.jpg",
					Format:      "JPEG",
					Title:       "2024 August After kleenex destruction",
					Description: "A fluffy Rosie, looking innocent after having shredded a kleenex lying nearby",
					CapturedAt:  "2024-08-04T05:42:02",
					Width:       4032,
					Height:      3024,
				},
			},
		},
		{
			name: "handles quotes in metadata",
			file: "quotes_in_metadata.jpg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "quotes_in_metadata.jpg",
					Format:      "JPEG",
					Title:       "AIgen We can chew it poster",
					Description: "A parody of the classic Rosie the Riveter poster, with a poodle in a red bandana saying \"We can chew it\"",
					CapturedAt:  "2026-05-01T20:42:44",
					Width:       1024,
					Height:      1535,
				},
			},
		},
		{
			name: "handles missing optional metadata",
			file: "partial_metadata.jpg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "partial_metadata.jpg",
					Format:      "JPEG",
					Title:       "",
					Description: "",
					CapturedAt:  "",
					Width:       1200,
					Height:      799,
				},
			},
		},
		{
			name: "passes through missing dimensions",
			file: "missing_metadata.jpg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "missing_metadata.jpg",
					Format:      "PDF",
					Title:       "plant fact sheet",
					Description: "",
					CapturedAt:  "",
					Width:       0,
					Height:      0,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := mustAbs(t, filepath.Join("testdata", tc.file))
			got, err := (&metadata.Exiftool{}).Read(path)
			if err != nil {
				t.Fatalf("Exiftool.Read(%q) returned error: %v", path, err)
			}
			if len(got.FileProblems) != 0 {
				t.Errorf("Exiftool.Read(%q) returned file problems: %v", path, got.FileProblems)
			}
			if !slices.Equal(got.Metadata, tc.wantMetadata) {
				t.Errorf("Exiftool.Read(%q) metadata mismatch\n got: %+v\nwant: %+v", path, got.Metadata, tc.wantMetadata)
			}
		})
	}
}

func TestExiftoolReadReadsDirectories(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		path := mustAbs(t, t.TempDir())
		got, err := (&metadata.Exiftool{}).Read(path)
		if err != nil {
			t.Fatalf("Exiftool.Read(%q) returned error: %v", path, err)
		}
		if len(got.Metadata) != 0 {
			t.Errorf("Exiftool.Read(%q) returned metadata: %+v", path, got.Metadata)
		}
		if len(got.FileProblems) != 0 {
			t.Errorf("Exiftool.Read(%q) returned file problems: %+v", path, got.FileProblems)
		}
	})

	t.Run("directory with good and bad files", func(t *testing.T) {
		dirPath := t.TempDir()
		copyFixture(t, dirPath, "complete_metadata.jpg")
		copyFixture(t, dirPath, "empty.jpg")

		wantMetadata := []image.Metadata{
			{
				FileName:    "complete_metadata.jpg",
				Format:      "JPEG",
				Title:       "2024 August After kleenex destruction",
				Description: "A fluffy Rosie, looking innocent after having shredded a kleenex lying nearby",
				CapturedAt:  "2024-08-04T05:42:02",
				Width:       4032,
				Height:      3024,
			},
		}
		wantProblems := []metadata.Problem{
			{
				FileName: "empty.jpg",
				Message:  "reported by exiftool: File is empty",
			},
		}

		path := mustAbs(t, dirPath)
		got, err := (&metadata.Exiftool{}).Read(path)
		if err != nil {
			t.Fatalf("Exiftool.Read(%q) returned error: %v", path, err)
		}
		if !slices.Equal(got.Metadata, wantMetadata) {
			t.Errorf("Exiftool.Read(%q) metadata mismatch\n got: %+v\nwant: %+v", path, got.Metadata, wantMetadata)
		}
		if !slices.Equal(got.FileProblems, wantProblems) {
			t.Errorf("Exiftool.Read(%q) file problems mismatch\n got: %+v\nwant: %+v", path, got.FileProblems, wantProblems)
		}
	})
}

func TestExiftoolReadReportsFileProblems(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		wantProblems []metadata.Problem
	}{
		{
			name: "errors reported by exiftool",
			file: "empty.jpg",
			wantProblems: []metadata.Problem{
				{
					FileName: "empty.jpg",
					Message:  "reported by exiftool: File is empty",
				},
			},
		},
		{
			name: "invalid DateTimeOriginal",
			file: "bad_datetimeoriginal.jpg",
			wantProblems: []metadata.Problem{
				{
					FileName: "bad_datetimeoriginal.jpg",
					Message:  `invalid DateTimeOriginal "2020-01-02T03:04:05": parsing time "2020-01-02T03:04:05" as "2006:01:02 15:04:05": cannot parse "-01-02T03:04:05" as ":"`,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := mustAbs(t, filepath.Join("testdata", tc.file))
			got, err := (&metadata.Exiftool{}).Read(path)
			if err != nil {
				t.Fatalf("Exiftool.Read(%q) returned error: %v", path, err)
			}
			if len(got.Metadata) != 0 {
				t.Errorf("Exiftool.Read(%q) returned metadata: %+v", path, got.Metadata)
			}
			if !slices.Equal(got.FileProblems, tc.wantProblems) {
				t.Errorf("Exiftool.Read(%q) file problems mismatch\n got: %+v\nwant: %+v", path, got.FileProblems, tc.wantProblems)
			}
		})
	}
}

func TestExiftoolReadReturnsError(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		wantMetadata []image.Metadata
	}{
		{
			name:         "missing file",
			file:         "nonexistent.jpg",
			wantMetadata: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := mustAbs(t, filepath.Join("testdata", tc.file))
			got, err := (&metadata.Exiftool{}).Read(path)
			if !slices.Equal(got.Metadata, tc.wantMetadata) {
				t.Errorf("Exiftool.Read(%q) metadata mismatch\n got: %+v\nwant: %+v", path, got.Metadata, tc.wantMetadata)
			}
			if err == nil {
				t.Fatalf("Exiftool.Read(%q) returned nil error, want error wrapping *exec.ExitError", path)
			}
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Errorf("Exiftool.Read(%q) returned error %v (%T), want error wrapping *exec.ExitError", path, err, err)
			}
		})
	}
}

func copyFixture(t *testing.T, dir, name string) {
	t.Helper()

	src := filepath.Join("testdata", name)
	dst := filepath.Join(dir, name)

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading fixture %q: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("writing fixture copy %q: %v", dst, err)
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
