package metadata

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
)

func TestFetchMetadata(t *testing.T) {
	var tests = map[string]struct {
		path            string
		desiredMetadata []image.Metadata
	}{
		"IMG_3916.jpeg": {
			path: filepath.Join("testdata", "IMG_3916.jpeg"),
			desiredMetadata: []image.Metadata{
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
		"riveter_chew_it_1024x1535.jpeg": {
			path: filepath.Join("testdata", "riveter_chew_it_1024x1535.jpeg"),
			desiredMetadata: []image.Metadata{
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
		"squash.jpg": {
			path: filepath.Join("testdata", "squash.jpg"),
			desiredMetadata: []image.Metadata{
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

	for testname, testdata := range tests {
		t.Run(testname, func(t *testing.T) {
			exiftool := Exiftool{}
			result, err := exiftool.Read(testdata.path)
			if err != nil {
				t.Errorf("Unexpected error returned when fetching metadata for %q: %v", testdata.path, err)
			}
			if len(result.FileProblems) != 0 {
				t.Errorf("Unexpected problems identified: %v", result.FileProblems)
			}
			if !slices.Equal(result.Metadata, testdata.desiredMetadata) {
				t.Errorf("Image metadata incorrect for %q.\n\n   Got: %+v\n\n   Wanted: %+v", testdata.path, result, testdata.desiredMetadata)
			}
		})
	}
}

func TestFetchMetadataExecError(t *testing.T) {
	var tests = map[string]struct {
		path            string
		desiredMetadata []image.Metadata
	}{
		"nonexistent.jpg": {
			path:            filepath.Join("testdata", "nonexistent.jpg"),
			desiredMetadata: nil,
		},
	}

	for testname, testdata := range tests {
		t.Run(testname, func(t *testing.T) {
			exiftool := Exiftool{}
			result, err := exiftool.Read(testdata.path)
			if !slices.Equal(result.Metadata, testdata.desiredMetadata) {
				t.Errorf("Image metadata incorrect for %q.\n\n   Got: %+v\n\n   Wanted: %+v", testdata.path, result, testdata.desiredMetadata)
			}
			if err == nil {
				t.Fatalf("Expected an error for %q.\n\n   Wanted 1 exec.ExitError but got none", testdata.path)
			}
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Errorf("Expected an exec.ExitError but got\n\n   %v\nwith type %T", err, err)
			}
		})
	}
}
