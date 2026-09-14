package gallery

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/manifest"
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

func TestNewTemplateData(t *testing.T) {
	tests := []struct {
		name      string
		urlPrefix string
		image     galleryImage
		want      templateData
	}{
		{
			name:      "builds picture data with sorted variants",
			urlPrefix: "/images",
			image: newGalleryImageWithManifest(t, manifest.Manifest{
				Title:       "Rosie title",
				Description: "Rosie alt text",
				Width:       1200,
				Height:      800,
				Variants: map[image.Format][]manifest.File{
					image.FormatJPEG: {
						{Path: mustRel(t, "w1200.jpg"), Width: 1200, Height: 800},
						{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 267},
						{Path: mustRel(t, "w800.jpg"), Width: 800, Height: 533},
					},
					image.FormatAVIF: {
						{Path: mustRel(t, "w800.avif"), Width: 800, Height: 533},
						{Path: mustRel(t, "w400.avif"), Width: 400, Height: 267},
					},
				},
			}),
			want: templateData{
				Images: []templateImage{
					{
						Sources: []templateSource{
							{
								Type:   "image/avif",
								Srcset: "/images/2024/05/abc123/w400.avif 400w, /images/2024/05/abc123/w800.avif 800w",
								Sizes:  imageSizes,
							},
						},
						Fallback: templateFallback{
							Src:    "/images/2024/05/abc123/w800.jpg",
							Srcset: "/images/2024/05/abc123/w400.jpg 400w, /images/2024/05/abc123/w800.jpg 800w, /images/2024/05/abc123/w1200.jpg 1200w",
							Sizes:  imageSizes,
							Width:  1200,
							Height: 800,
							Alt:    "Rosie alt text",
						},
					},
				},
			},
		},
		{
			name:      "omits AVIF source when AVIF variants are missing",
			urlPrefix: "",
			image: newGalleryImageWithManifest(t, manifest.Manifest{
				Title:  "Rosie title",
				Width:  400,
				Height: 267,
				Variants: map[image.Format][]manifest.File{
					image.FormatJPEG: {
						{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 267},
					},
				},
			}),
			want: templateData{
				Images: []templateImage{
					{
						Fallback: templateFallback{
							Src:    "2024/05/abc123/w400.jpg",
							Srcset: "2024/05/abc123/w400.jpg 400w",
							Sizes:  imageSizes,
							Width:  400,
							Height: 267,
							Alt:    "Rosie title",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newTemplateData([]galleryImage{tc.image}, tc.urlPrefix)
			if err != nil {
				t.Fatalf("newTemplateData() returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("newTemplateData() mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestNewTemplateDataReturnsErrorForMissingJPEG(t *testing.T) {
	img := newGalleryImageWithManifest(t, manifest.Manifest{
		Variants: map[image.Format][]manifest.File{
			image.FormatAVIF: {
				{Path: mustRel(t, "w400.avif"), Width: 400, Height: 267},
			},
		},
	})

	_, err := newTemplateData([]galleryImage{img}, "/images")
	if err == nil {
		t.Fatalf("newTemplateData() returned nil error, want error")
	}
	wantMessage := `image "2024/05/abc123": missing JPEG variants`
	if !strings.Contains(err.Error(), wantMessage) {
		t.Errorf("newTemplateData() error = %q, want containing %q", err, wantMessage)
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

func newGalleryImageWithManifest(t *testing.T, m manifest.Manifest) galleryImage {
	t.Helper()

	img := newGalleryImage(t, "2024/05/abc123", "2024-05-12T14:22:00", "2026-08-24T18:00:00Z")
	img.manifest = m
	return img
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
