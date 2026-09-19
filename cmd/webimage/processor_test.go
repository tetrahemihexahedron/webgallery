package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/paths"
	"tetrahemihexahedron/webimage/internal/variants"
)

const fixtureSHA256 = "9d105ded7ef2002873fbe783fb9727f44100c6891cfad76b832eaa55be77f3af"

func TestProcessIncomingDir(t *testing.T) {
	p := newIntegrationProcessor(t)

	got, err := p.processIncomingDir()
	if err != nil {
		t.Fatalf("imageProcessor.processIncomingDir() returned error: %v", err)
	}
	if len(got.problems) != 0 {
		t.Errorf("imageProcessor.processIncomingDir() returned problems: %+v", got.problems)
	}
	if len(got.images) != 1 {
		t.Fatalf("imageProcessor.processIncomingDir() returned %d images, want 1: %+v", len(got.images), got.images)
	}

	processed := got.images[0]
	wantSource := image.Source{Hash: fixtureSHA256, Width: 800, Height: 1067}
	if processed.Source != wantSource {
		t.Errorf("imageProcessor.processIncomingDir() source = %+v, want %+v", processed.Source, wantSource)
	}
	if processed.Title != "2023 October Posing" {
		t.Errorf("imageProcessor.processIncomingDir() title = %q, want %q", processed.Title, "2023 October Posing")
	}
	if processed.Description != "Rosie as a small puppy, sitting and looking directly at the camera." {
		t.Errorf("imageProcessor.processIncomingDir() description = %q, want fixture description", processed.Description)
	}
	if processed.CapturedAt != "2023-10-03T17:26:39" {
		t.Errorf("imageProcessor.processIncomingDir() capturedAt = %q, want %q", processed.CapturedAt, "2023-10-03T17:26:39")
	}
	if _, err := image.ParseProcessedAt(processed.ProcessedAt); err != nil {
		t.Errorf("imageProcessor.processIncomingDir() processedAt = %q, want valid processed datetime: %v", processed.ProcessedAt, err)
	}
	if got := filepath.Dir(processed.DirRelPath.String()); got != "2023/10" {
		t.Errorf("imageProcessor.processIncomingDir() image directory parent = %q, want %q", got, "2023/10")
	}

	wantVariants := []image.Variant{
		{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400},
		{Path: mustRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800},
		{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400},
		{Path: mustRel(t, "w800.avif"), Format: image.FormatAVIF, Width: 800},
	}
	if !slices.Equal(processed.Variants, wantVariants) {
		t.Errorf("imageProcessor.processIncomingDir() variants mismatch\n got: %+v\nwant: %+v", processed.Variants, wantVariants)
	}

	imageDir, err := paths.JoinAbs(p.cfg.OutDir, processed.DirRelPath)
	if err != nil {
		t.Fatalf("paths.JoinAbs(%q, %q) returned error: %v", p.cfg.OutDir, processed.DirRelPath, err)
	}
	assertOutputFiles(t, p.cfg.InDir, imageDir)
	assertManifest(t, imageDir, processed)
	assertIndex(t, p.cfg.OutDir, processed)
}

func TestProcessImageRejectsIncompleteVariants(t *testing.T) {
	variantErr := errors.New("variant generation failed")
	tests := []struct {
		name        string
		result      func([]variants.Spec) variants.Result
		wantProblem string
		wantErr     error
	}{
		{
			name: "failed variant",
			result: func(specs []variants.Spec) variants.Result {
				generatedJPEG := specs[0]
				failedAVIF := specs[2]
				return variants.Result{
					Generated: []variants.Spec{generatedJPEG},
					Failed: []variants.Failure{
						{Spec: failedAVIF, Err: variantErr},
					},
				}
			},
			wantProblem: "1 generated, 1 failed",
			wantErr:     variantErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var imageDir string
			generator := fakeVariantGenerator(func(_ paths.AbsPath, specs []variants.Spec) (variants.Result, error) {
				imageDir = filepath.Dir(specs[0].OutPath.String())
				return tc.result(specs), nil
			})
			p := imageProcessor{
				cfg: Config{
					OutDir:  mustAbs(t, t.TempDir()),
					DirDate: DirDateProcessed,
				},
				variantGenerator: generator,
			}
			source := sourceImage{
				metadata: image.Metadata{Width: 800, Height: 1067},
				path:     mustAbs(t, filepath.Join("testdata", "incoming", "image_800x1067.jpg")),
				sha256:   fixtureSHA256,
			}

			_, err := p.processImage(source)
			if err == nil {
				t.Fatal("imageProcessor.processImage() returned nil error, want error")
			}
			if !strings.Contains(err.Error(), tc.wantProblem) {
				t.Errorf("imageProcessor.processImage() error = %v, want message containing %q", err, tc.wantProblem)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("imageProcessor.processImage() error = %v, want error wrapping %v", err, tc.wantErr)
			}
			if imageDir == "" {
				t.Fatal("fakeVariantGenerator did not receive variant specifications")
			}
			if _, statErr := os.Stat(imageDir); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("os.Stat(%q) error = %v, want os.ErrNotExist", imageDir, statErr)
			}
		})
	}
}

func TestProcessIncomingDirSkipsPreviouslyProcessedImage(t *testing.T) {
	p := newIntegrationProcessor(t)

	if _, err := p.processIncomingDir(); err != nil {
		t.Fatalf("first imageProcessor.processIncomingDir() returned error: %v", err)
	}
	indexBefore, err := index.ReadDir(p.cfg.OutDir)
	if err != nil {
		t.Fatalf("index.ReadDir(%q) after first processIncomingDir returned error: %v", p.cfg.OutDir, err)
	}

	got, err := p.processIncomingDir()
	if err != nil {
		t.Fatalf("second imageProcessor.processIncomingDir() returned error: %v", err)
	}
	if len(got.images) != 0 {
		t.Errorf("second imageProcessor.processIncomingDir() returned images: %+v, want none", got.images)
	}
	if len(got.problems) != 1 {
		t.Fatalf("second imageProcessor.processIncomingDir() returned %d problems, want 1: %+v", len(got.problems), got.problems)
	}
	problem := got.problems[0]
	if problem.fileName != "image_800x1067.jpg" {
		t.Errorf("duplicate problem fileName = %q, want %q", problem.fileName, "image_800x1067.jpg")
	}
	if !strings.Contains(problem.message, "duplicate") {
		t.Errorf("duplicate problem message = %q, want message containing %q", problem.message, "duplicate")
	}

	indexAfter, err := index.ReadDir(p.cfg.OutDir)
	if err != nil {
		t.Fatalf("index.ReadDir(%q) after second processIncomingDir returned error: %v", p.cfg.OutDir, err)
	}
	if !reflect.DeepEqual(indexAfter, indexBefore) {
		t.Errorf("index after duplicate processing = %+v, want unchanged index %+v", indexAfter, indexBefore)
	}
}

type fakeVariantGenerator func(paths.AbsPath, []variants.Spec) (variants.Result, error)

func (f fakeVariantGenerator) Generate(source paths.AbsPath, specs []variants.Spec) (variants.Result, error) {
	return f(source, specs)
}

func newIntegrationProcessor(t *testing.T) *imageProcessor {
	t.Helper()

	return &imageProcessor{
		cfg: Config{
			InDir:   mustAbs(t, filepath.Join("testdata", "incoming")),
			OutDir:  mustAbs(t, t.TempDir()),
			DirDate: DirDateCaptured,
		},
		metadataReader:   &metadata.Exiftool{},
		variantGenerator: &variants.Vipsthumbnail{},
		progressReporter: io.Discard,
	}
}

func assertOutputFiles(t *testing.T, incomingDir, imageDir paths.AbsPath) {
	t.Helper()

	entries, err := os.ReadDir(imageDir.String())
	if err != nil {
		t.Fatalf("os.ReadDir(%q) returned error: %v", imageDir, err)
	}
	gotNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		gotNames = append(gotNames, entry.Name())
	}
	wantNames := []string{"manifest.json", "orig.jpg", "w400.avif", "w400.jpg", "w800.avif", "w800.jpg"}
	if !slices.Equal(gotNames, wantNames) {
		t.Errorf("output files in %q mismatch\n got: %v\nwant: %v", imageDir, gotNames, wantNames)
	}

	source, err := os.ReadFile(filepath.Join(incomingDir.String(), "image_800x1067.jpg"))
	if err != nil {
		t.Fatalf("reading source fixture: %v", err)
	}
	original, err := os.ReadFile(filepath.Join(imageDir.String(), "orig.jpg"))
	if err != nil {
		t.Fatalf("reading copied original: %v", err)
	}
	if !bytes.Equal(original, source) {
		t.Error("copied orig.jpg differs from source fixture")
	}

	for _, name := range wantNames[2:] {
		info, err := os.Stat(filepath.Join(imageDir.String(), name))
		if err != nil {
			t.Errorf("os.Stat(%q) returned error: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("generated variant %q is empty", name)
		}
	}
}

func assertManifest(t *testing.T, imageDir paths.AbsPath, processed image.Processed) {
	t.Helper()

	path, err := manifest.ManifestPath(imageDir)
	if err != nil {
		t.Fatalf("manifest.ManifestPath(%q) returned error: %v", imageDir, err)
	}
	got, err := manifest.ReadFile(path)
	if err != nil {
		t.Fatalf("manifest.ReadFile(%q) returned error: %v", path, err)
	}
	want := manifest.Manifest{
		Title:       processed.Title,
		Description: processed.Description,
		CapturedAt:  processed.CapturedAt,
		ProcessedAt: processed.ProcessedAt,
		SHA256:      fixtureSHA256,
		Width:       800,
		Height:      1067,
		Variants: map[image.Format][]manifest.VariantFile{
			image.FormatJPEG: {
				{Path: mustRel(t, "w400.jpg"), Width: 400, Height: 534},
				{Path: mustRel(t, "w800.jpg"), Width: 800, Height: 1067},
			},
			image.FormatAVIF: {
				{Path: mustRel(t, "w400.avif"), Width: 400, Height: 534},
				{Path: mustRel(t, "w800.avif"), Width: 800, Height: 1067},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("written manifest mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func assertIndex(t *testing.T, outDir paths.AbsPath, processed image.Processed) {
	t.Helper()

	got, err := index.ReadDir(outDir)
	if err != nil {
		t.Fatalf("index.ReadDir(%q) returned error: %v", outDir, err)
	}
	if _, err := image.ParseProcessedAt(got.GeneratedAt); err != nil {
		t.Errorf("index generatedAt = %q, want valid processed datetime: %v", got.GeneratedAt, err)
	}
	if len(got.Images) != 1 {
		t.Fatalf("index contains %d images, want 1: %+v", len(got.Images), got.Images)
	}

	want := index.Entry{
		Dir:         processed.DirRelPath,
		Manifest:    mustRel(t, filepath.Join(processed.DirRelPath.String(), "manifest.json")),
		Title:       processed.Title,
		CapturedAt:  processed.CapturedAt,
		ProcessedAt: processed.ProcessedAt,
		SHA256:      fixtureSHA256,
	}
	if got.Images[0] != want {
		t.Errorf("index image = %+v, want %+v", got.Images[0], want)
	}
}
