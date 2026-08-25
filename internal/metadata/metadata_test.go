package metadata

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.file)
			got, err := (&Exiftool{}).Read(path)
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

func TestExiftoolReadReportsFileProblems(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		wantProblems []Problem
	}{
		{
			name: "errors reported by exiftool",
			file: "empty.jpg",
			wantProblems: []Problem{
				{
					FileName: "empty.jpg",
					Message:  "reported by exiftool: File is empty",
				},
			},
		},
		{
			name: "missing required metadata",
			file: "missing_metadata.jpg",
			wantProblems: []Problem{
				{
					FileName: "missing_metadata.jpg",
					Message:  "missing required metadata: [Width Height]",
				},
			},
		},
		{
			name: "invalid DateTimeOriginal",
			file: "bad_datetimeoriginal.jpg",
			wantProblems: []Problem{
				{
					FileName: "bad_datetimeoriginal.jpg",
					Message:  `invalid DateTimeOriginal "2020-01-02T03:04:05": parsing time "2020-01-02T03:04:05" as "2006:01:02 15:04:05": cannot parse "-01-02T03:04:05" as ":"`,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.file)
			got, err := (&Exiftool{}).Read(path)
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
			path := filepath.Join("testdata", tc.file)
			got, err := (&Exiftool{}).Read(path)
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
