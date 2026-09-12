package gallery

import (
	"slices"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/paths"
)

func TestSortImages(t *testing.T) {
	tests := []struct {
		name           string
		sortField      SortField
		images         []galleryImage
		wantSortedDirs []string
	}{
		{
			name:      "captured newest first",
			sortField: SortCaptured,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/b", "2024-05-12T14:22:00", "2026-08-24T18:00:00Z"),
				newGalleryImage(t, "2024/05/a", "2024-05-13T14:22:00", "2026-08-23T18:00:00Z"),
				newGalleryImage(t, "2024/05/c", "2024-05-11T14:22:00", "2026-08-25T18:00:00Z"),
			},
			wantSortedDirs: []string{"2024/05/a", "2024/05/b", "2024/05/c"},
		},
		{
			name:      "processed newest first",
			sortField: SortProcessed,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/b", "2024-05-12T14:22:00", "2026-08-24T18:00:00Z"),
				newGalleryImage(t, "2024/05/a", "2024-05-13T14:22:00", "2026-08-23T18:00:00Z"),
				newGalleryImage(t, "2024/05/c", "2024-05-11T14:22:00", "2026-08-25T18:00:00Z"),
			},
			wantSortedDirs: []string{"2024/05/c", "2024/05/b", "2024/05/a"},
		},
		{
			name:      "ties sort by directory",
			sortField: SortCaptured,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/b", "2024-05-12T14:22:00", "2026-08-24T18:00:00Z"),
				newGalleryImage(t, "2024/05/a", "2024-05-12T14:22:00", "2026-08-23T18:00:00Z"),
			},
			wantSortedDirs: []string{"2024/05/a", "2024/05/b"},
		},
		{
			name:      "processed sort allows empty capturedAt",
			sortField: SortProcessed,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/a", "", "2026-08-23T18:00:00Z"),
				newGalleryImage(t, "2024/05/b", "", "2026-08-24T18:00:00Z"),
			},
			wantSortedDirs: []string{"2024/05/b", "2024/05/a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := slices.Clone(tc.images)
			if err := sortImages(got, tc.sortField); err != nil {
				t.Fatalf("sortImages() returned error: %v", err)
			}

			gotSortedDirs := imageDirs(got)
			if !slices.Equal(gotSortedDirs, tc.wantSortedDirs) {
				t.Errorf("sortImages() dirs = %+v, want %+v", gotSortedDirs, tc.wantSortedDirs)
			}
		})
	}
}

func TestSortImagesReturnsError(t *testing.T) {
	tests := []struct {
		name        string
		sortField   SortField
		images      []galleryImage
		wantMessage string
	}{
		{
			name:      "missing capturedAt",
			sortField: SortCaptured,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/a", "", "2026-08-24T18:00:00Z"),
			},
			wantMessage: `image "2024/05/a": invalid capturedAt`,
		},
		{
			name:      "invalid processedAt",
			sortField: SortProcessed,
			images: []galleryImage{
				newGalleryImage(t, "2024/05/a", "2024-05-12T14:22:00", "2026-08-24T18:00:00"),
			},
			wantMessage: `image "2024/05/a": invalid processedAt`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := sortImages(tc.images, tc.sortField); err == nil {
				t.Fatalf("sortImages() returned nil error, want error")
			} else if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("sortImages() error = %q, want containing %q", err, tc.wantMessage)
			}
		})
	}
}

func newGalleryImage(t *testing.T, dir string, capturedAt string, processedAt string) galleryImage {
	t.Helper()

	return galleryImage{
		indexImage: index.Image{
			Dir:         mustRel(t, dir),
			CapturedAt:  capturedAt,
			ProcessedAt: processedAt,
		},
	}
}

func imageDirs(images []galleryImage) []string {
	dirs := make([]string, 0, len(images))
	for _, img := range images {
		dirs = append(dirs, img.indexImage.Dir.String())
	}
	return dirs
}

func mustRel(t *testing.T, path string) paths.RelPath {
	t.Helper()

	p, err := paths.NewRelPath(path)
	if err != nil {
		t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", path, err)
	}

	return p
}
