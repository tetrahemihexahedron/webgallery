package gallery

import (
	"bytes"
	"os"
	"path/filepath"
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
		modify         func([]galleryImage)
		wantSortedDirs []string
	}{
		{
			name:           "captured newest first",
			sortField:      SortCaptured,
			wantSortedDirs: []string{"2024/05/abc123", "2024/05/bbb222", "2024/05/ccc333"},
		},
		{
			name:           "processed newest first",
			sortField:      SortProcessed,
			wantSortedDirs: []string{"2024/05/ccc333", "2024/05/bbb222", "2024/05/abc123"},
		},
		{
			name:      "ties sort by directory",
			sortField: SortCaptured,
			modify: func(images []galleryImage) {
				for i := range images {
					images[i].indexImage.CapturedAt = "2024-05-12T14:22:00"
				}
			},
			wantSortedDirs: []string{"2024/05/abc123", "2024/05/bbb222", "2024/05/ccc333"},
		},
		{
			name:      "processed sort allows empty capturedAt",
			sortField: SortProcessed,
			modify: func(images []galleryImage) {
				for i := range images {
					images[i].indexImage.CapturedAt = ""
				}
			},
			wantSortedDirs: []string{"2024/05/ccc333", "2024/05/bbb222", "2024/05/abc123"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newBaseImages(t)
			if tc.modify != nil {
				tc.modify(got)
			}

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

func TestRender(t *testing.T) {
	var got bytes.Buffer
	if err := Render(&got, Options{ImagesRoot: testdataPath(t, "render"), URLPrefix: "/images", Sort: SortCaptured}); err != nil {
		t.Fatalf("Render() returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "render.html"))
	if err != nil {
		t.Fatalf("reading expected render output: %v", err)
	}

	if got.String() != string(want) {
		t.Errorf("Render() output mismatch\n got:\n%s\nwant:\n%s", got.String(), string(want))
	}
}

func TestRenderReturnsErrorForMissingIndex(t *testing.T) {
	var got bytes.Buffer
	err := Render(&got, Options{ImagesRoot: mustAbs(t, t.TempDir()), Sort: SortCaptured})
	if err == nil {
		t.Fatalf("Render() returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "reading gallery index") {
		t.Errorf("Render() error = %q, want containing %q", err, "reading gallery index")
	}
}

func TestNewTemplateData(t *testing.T) {
	tests := []struct {
		name      string
		urlPrefix string
		modify    func(*galleryImage)
		want      templateData
	}{
		{
			name:      "builds picture data with sorted variants",
			urlPrefix: "/images",
			want: templateData{Images: []templateImage{{
				Sources: []templateSource{{
					Type:   "image/avif",
					Srcset: "/images/2024/05/abc123/w400.avif 400w, /images/2024/05/abc123/w800.avif 800w",
					Sizes:  imageSizes,
				}},
				Fallback: templateFallback{
					Src:    "/images/2024/05/abc123/w800.jpg",
					Srcset: "/images/2024/05/abc123/w400.jpg 400w, /images/2024/05/abc123/w800.jpg 800w, /images/2024/05/abc123/w1200.jpg 1200w",
					Sizes:  imageSizes,
					Width:  1200,
					Height: 800,
					Alt:    "Rosie alt text",
				},
			}}},
		},
		{
			name:      "omits AVIF source when AVIF variants are missing",
			urlPrefix: "",
			modify: func(img *galleryImage) {
				delete(img.manifest.Variants, image.FormatAVIF)
			},
			want: templateData{Images: []templateImage{{
				Fallback: templateFallback{
					Src:    "2024/05/abc123/w800.jpg",
					Srcset: "2024/05/abc123/w400.jpg 400w, 2024/05/abc123/w800.jpg 800w, 2024/05/abc123/w1200.jpg 1200w",
					Sizes:  imageSizes,
					Width:  1200,
					Height: 800,
					Alt:    "Rosie alt text",
				},
			}}},
		},
		{
			name:      "uses title for alt text when description is empty",
			urlPrefix: "/images",
			modify: func(img *galleryImage) {
				img.manifest.Description = ""
			},
			want: templateData{Images: []templateImage{{
				Sources: []templateSource{{
					Type:   "image/avif",
					Srcset: "/images/2024/05/abc123/w400.avif 400w, /images/2024/05/abc123/w800.avif 800w",
					Sizes:  imageSizes,
				}},
				Fallback: templateFallback{
					Src:    "/images/2024/05/abc123/w800.jpg",
					Srcset: "/images/2024/05/abc123/w400.jpg 400w, /images/2024/05/abc123/w800.jpg 800w, /images/2024/05/abc123/w1200.jpg 1200w",
					Sizes:  imageSizes,
					Width:  1200,
					Height: 800,
					Alt:    "Rosie title",
				},
			}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotImage := newBaseImages(t)[0]
			if tc.modify != nil {
				tc.modify(&gotImage)
			}

			got, err := newTemplateData([]galleryImage{gotImage}, tc.urlPrefix)
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
	img := newBaseImages(t)[0]
	delete(img.manifest.Variants, image.FormatJPEG)

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
		modify      func(*galleryImage)
		wantMessage string
	}{
		{
			name:      "missing capturedAt",
			sortField: SortCaptured,
			modify: func(img *galleryImage) {
				img.indexImage.CapturedAt = ""
			},
			wantMessage: `image "2024/05/abc123": invalid capturedAt`,
		},
		{
			name:      "invalid processedAt",
			sortField: SortProcessed,
			modify: func(img *galleryImage) {
				img.indexImage.ProcessedAt = "2026-08-24T18:00:00"
			},
			wantMessage: `image "2024/05/abc123": invalid processedAt`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := newBaseImages(t)[0]
			if tc.modify != nil {
				tc.modify(&img)
			}

			if err := sortImages([]galleryImage{img}, tc.sortField); err == nil {
				t.Fatalf("sortImages() returned nil error, want error")
			} else if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("sortImages() error = %q, want containing %q", err, tc.wantMessage)
			}
		})
	}
}

func newBaseImages(t *testing.T) []galleryImage {
	t.Helper()

	newImage := func(dir, title, description, capturedAt, processedAt, sha256 string) galleryImage {
		return galleryImage{
			indexImage: index.Image{
				Dir:         mustRel(t, dir),
				Manifest:    mustRel(t, dir+"/manifest.json"),
				Title:       title,
				CapturedAt:  capturedAt,
				ProcessedAt: processedAt,
				SHA256:      sha256,
			},
			manifest: manifest.Manifest{
				Title:       title,
				Description: description,
				CapturedAt:  capturedAt,
				ProcessedAt: processedAt,
				SHA256:      sha256,
				Width:       1200,
				Height:      800,
				Variants: map[image.Format][]manifest.VariantFile{
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
			},
		}
	}

	return []galleryImage{
		newImage("2024/05/abc123", "Rosie title", "Rosie alt text", "2024-05-13T14:22:00", "2026-08-23T18:00:00Z", "abc123"),
		newImage("2024/05/bbb222", "Riveter title", "Riveter alt text", "2024-05-12T14:22:00", "2026-08-24T18:00:00Z", "bbb222"),
		newImage("2024/05/ccc333", "Poster title", "Poster alt text", "2024-05-11T14:22:00", "2026-08-25T18:00:00Z", "ccc333"),
	}
}

func imageDirs(images []galleryImage) []string {
	dirs := make([]string, 0, len(images))
	for _, img := range images {
		dirs = append(dirs, img.indexImage.Dir.String())
	}
	return dirs
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
