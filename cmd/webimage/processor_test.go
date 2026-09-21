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
		{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400, Height: 534},
		{Path: mustRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800, Height: 1067},
		{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400, Height: 534},
		{Path: mustRel(t, "w800.avif"), Format: image.FormatAVIF, Width: 800, Height: 1067},
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

func TestProcessMetadataEntryValidatesJPEGMetadata(t *testing.T) {
	sourceContents, err := os.ReadFile(filepath.Join("testdata", "incoming", "image_800x1067.jpg"))
	if err != nil {
		t.Fatalf("reading source fixture: %v", err)
	}

	validMetadata := image.Metadata{
		FileName:   "image.jpg",
		Format:     image.FormatJPEG.String(),
		CapturedAt: "2024-05-12T14:22:00",
		Width:      800,
		Height:     1067,
	}
	tests := []struct {
		name          string
		dirDate       DirDate
		change        func(*image.Metadata)
		createSource  bool
		wantProblem   bool
		wantMessage   string
		wantGenerator bool
	}{
		{
			name:    "zero width",
			dirDate: DirDateProcessed,
			change: func(meta *image.Metadata) {
				meta.Width = 0
			},
			wantProblem: true,
			wantMessage: "width must be positive",
		},
		{
			name:    "negative height",
			dirDate: DirDateProcessed,
			change: func(meta *image.Metadata) {
				meta.Height = -1
			},
			wantProblem: true,
			wantMessage: "height must be positive",
		},
		{
			name:    "empty capture date with processed directories",
			dirDate: DirDateProcessed,
			change: func(meta *image.Metadata) {
				meta.CapturedAt = ""
			},
			createSource:  true,
			wantGenerator: true,
		},
		{
			name:    "empty capture date with captured directories",
			dirDate: DirDateCaptured,
			change: func(meta *image.Metadata) {
				meta.CapturedAt = ""
			},
			wantProblem: true,
			wantMessage: "capturedAt is required",
		},
		{
			name:    "malformed capture date",
			dirDate: DirDateProcessed,
			change: func(meta *image.Metadata) {
				meta.CapturedAt = "2024:05:12 14:22:00"
			},
			wantProblem: true,
			wantMessage: "invalid capturedAt",
		},
		{
			name:    "noncanonical capture date",
			dirDate: DirDateProcessed,
			change: func(meta *image.Metadata) {
				meta.CapturedAt = " 2024-05-12T14:22:00"
			},
			wantProblem: true,
			wantMessage: "invalid capturedAt",
		},
		{
			name:    "unsupported format checked first",
			dirDate: DirDateCaptured,
			change: func(meta *image.Metadata) {
				meta.Format = "PNG"
				meta.Width = 0
				meta.Height = 0
			},
			wantProblem: true,
			wantMessage: "not JPEG",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			incomingDir := t.TempDir()
			outDir := t.TempDir()
			meta := validMetadata
			tc.change(&meta)
			if tc.createSource {
				sourcePath := filepath.Join(incomingDir, meta.FileName)
				if err := os.WriteFile(sourcePath, sourceContents, 0644); err != nil {
					t.Fatalf("os.WriteFile(%q) returned error: %v", sourcePath, err)
				}
			}

			generatorCalled := false
			variantPath := mustRel(t, "w800.jpg")
			generator := variantGenerator(func(req variants.Request) (variants.Result, error) {
				generatorCalled = true
				outputPath := filepath.Join(req.OutputDir.String(), variantPath.String())
				if err := os.WriteFile(outputPath, sourceContents, 0644); err != nil {
					return variants.Result{}, err
				}
				return variants.Result{Generated: []image.Variant{
					{Path: variantPath, Format: image.FormatJPEG, Width: 800},
				}}, nil
			})
			p := imageProcessor{
				cfg: Config{
					InDir:   mustAbs(t, incomingDir),
					OutDir:  mustAbs(t, outDir),
					DirDate: tc.dirDate,
				},
				variantGenerator: generator,
				progressReporter: io.Discard,
			}

			_, problem := p.processMetadataEntry(meta, map[string]paths.RelPath{})
			if got := problem != nil; got != tc.wantProblem {
				t.Fatalf("imageProcessor.processMetadataEntry() returned problem = %t, want %t: %+v", got, tc.wantProblem, problem)
			}
			if problem != nil {
				if problem.fileName != meta.FileName {
					t.Errorf("problem fileName = %q, want %q", problem.fileName, meta.FileName)
				}
				if !strings.Contains(problem.message, tc.wantMessage) {
					t.Errorf("problem message = %q, want message containing %q", problem.message, tc.wantMessage)
				}
			}
			if generatorCalled != tc.wantGenerator {
				t.Errorf("variant generator called = %t, want %t", generatorCalled, tc.wantGenerator)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	sourceContents := []byte("source image contents")

	t.Run("new destination", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := filepath.Join(dir, "source.jpg")
		destPath := filepath.Join(dir, "destination.jpg")
		if err := os.WriteFile(sourcePath, sourceContents, 0600); err != nil {
			t.Fatalf("os.WriteFile(%q) returned error: %v", sourcePath, err)
		}

		if err := copyFile(mustAbs(t, sourcePath), mustAbs(t, destPath)); err != nil {
			t.Fatalf("copyFile(%q, %q) returned error: %v", sourcePath, destPath, err)
		}
		got, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) returned error: %v", destPath, err)
		}
		if !bytes.Equal(got, sourceContents) {
			t.Errorf("copyFile(%q, %q) contents = %q, want %q", sourcePath, destPath, got, sourceContents)
		}
	})

	existingContents := []byte("existing destination contents")
	tests := []struct {
		name        string
		prepareDest func(sourcePath, destPath string) (string, error)
		wantDest    []byte
	}{
		{
			name: "same path",
			prepareDest: func(sourcePath, _ string) (string, error) {
				return sourcePath, nil
			},
			wantDest: sourceContents,
		},
		{
			name: "hard-link alias",
			prepareDest: func(sourcePath, destPath string) (string, error) {
				return destPath, os.Link(sourcePath, destPath)
			},
			wantDest: sourceContents,
		},
		{
			name: "symlink alias",
			prepareDest: func(sourcePath, destPath string) (string, error) {
				return destPath, os.Symlink(sourcePath, destPath)
			},
			wantDest: sourceContents,
		},
		{
			name: "unrelated pre-existing destination",
			prepareDest: func(_, destPath string) (string, error) {
				return destPath, os.WriteFile(destPath, existingContents, 0644)
			},
			wantDest: existingContents,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sourcePath := filepath.Join(dir, "source.jpg")
			if err := os.WriteFile(sourcePath, sourceContents, 0600); err != nil {
				t.Fatalf("os.WriteFile(%q) returned error: %v", sourcePath, err)
			}
			destPath, err := tc.prepareDest(sourcePath, filepath.Join(dir, "destination.jpg"))
			if err != nil {
				t.Fatalf("preparing destination: %v", err)
			}

			err = copyFile(mustAbs(t, sourcePath), mustAbs(t, destPath))
			if err == nil {
				t.Fatalf("copyFile(%q, %q) returned nil error, want error", sourcePath, destPath)
			}

			gotSource, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) returned error: %v", sourcePath, err)
			}
			if !bytes.Equal(gotSource, sourceContents) {
				t.Errorf("source contents after copyFile() = %q, want %q", gotSource, sourceContents)
			}
			gotDest, err := os.ReadFile(destPath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) returned error: %v", destPath, err)
			}
			if !bytes.Equal(gotDest, tc.wantDest) {
				t.Errorf("destination contents after copyFile() = %q, want %q", gotDest, tc.wantDest)
			}
		})
	}
}

func TestProcessImageRejectsIncompleteVariants(t *testing.T) {
	variantErr := errors.New("variant generation failed")
	tests := []struct {
		name        string
		result      variants.Result
		wantProblem string
		wantErr     error
	}{
		{
			name: "failed variant",
			result: variants.Result{
				Generated: []image.Variant{
					{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400},
				},
				Failed: []variants.Failure{
					{Format: image.FormatAVIF, Width: 400, Err: variantErr},
				},
			},
			wantProblem: "1 generated, 1 failed",
			wantErr:     variantErr,
		},
		{
			name: "no JPEG fallback",
			result: variants.Result{Generated: []image.Variant{
				{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400},
			}},
			wantProblem: "no JPEG fallback was generated",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var imageDir string
			generator := variantGenerator(func(req variants.Request) (variants.Result, error) {
				imageDir = req.OutputDir.String()
				return tc.result, nil
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
				t.Fatal("variantGenerator did not receive a request")
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

func newIntegrationProcessor(t *testing.T) *imageProcessor {
	t.Helper()

	return &imageProcessor{
		cfg: Config{
			InDir:   mustAbs(t, filepath.Join("testdata", "incoming")),
			OutDir:  mustAbs(t, t.TempDir()),
			DirDate: DirDateCaptured,
		},
		metadataReader:   &metadata.Exiftool{},
		variantGenerator: variants.Generate,
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
