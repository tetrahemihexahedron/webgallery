package exif

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
					Width:       1200,
					Height:      799,
				},
			},
		},
	}

	for testname, testdata := range tests {
		t.Run(testname, func(t *testing.T) {
			metadata, errs := FetchMetadata(testdata.path)
			if len(errs) != 0 {
				t.Errorf("Unexpected errors returned when fetching metadata for %q: %v", testdata.path, errs)
			}
			if !slices.Equal(metadata, testdata.desiredMetadata) {
				t.Errorf("Image metadata incorrect for %q.\n\n   Got: %+v\n\n   Wanted: %+v", testdata.path, metadata, testdata.desiredMetadata)
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
			metadata, errs := FetchMetadata(testdata.path)
			if !slices.Equal(metadata, testdata.desiredMetadata) {
				t.Errorf("Image metadata incorrect for %q.\n\n   Got: %+v\n\n   Wanted: %+v", testdata.path, metadata, testdata.desiredMetadata)
			}
			if len(errs) != 1 {
				t.Fatalf("Errors incorrect for %q.\n\n   Got: %+v\n\n   Wanted 1 exec.ExitError", testdata.path, errs)
			}
			err := errs[0]
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Errorf("Expected an exec.ExitError but got\n\n   %v\nwith type %T", err, err)
			}
		})
	}
}
