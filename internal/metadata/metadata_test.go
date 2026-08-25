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
			file: "IMG_3916.jpeg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "IMG_3916.jpeg",
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
			file: "riveter_chew_it_1024x1535.jpeg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "riveter_chew_it_1024x1535.jpeg",
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
			file: "squash.jpg",
			wantMetadata: []image.Metadata{
				{
					FileName:    "squash.jpg",
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
			exiftool := Exiftool{}
			got, err := exiftool.Read(path)
			if err != nil {
				t.Errorf("Unexpected error returned: %v", err)
			}
			if len(got.FileProblems) != 0 {
				t.Errorf("Unexpected problems identified: %v", got.FileProblems)
			}
			if !slices.Equal(got.Metadata, tc.wantMetadata) {
				t.Errorf("Image metadata incorrect.\n\n   Got: %+v\n\n   Want: %+v", got.Metadata, tc.wantMetadata)
			}
		})
	}
}

func TestExiftoolReadReturnsErrorForMissingFile(t *testing.T) {
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
			exiftool := Exiftool{}
			got, err := exiftool.Read(path)
			if !slices.Equal(got.Metadata, tc.wantMetadata) {
				t.Errorf("Image metadata incorrect.\n\n   Got: %+v\n\n   Want: %+v", got, tc.wantMetadata)
			}
			if err == nil {
				t.Fatal("Want an exec.ExitError but got no errors")
			}
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Errorf("Want an exec.ExitError but got: %#v", err)
			}
		})
	}
}
