package variants_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"tetrahemihexahedron/webgallery/internal/image"
	"tetrahemihexahedron/webgallery/internal/paths"
	"tetrahemihexahedron/webgallery/internal/variants"
)

type imageSize struct {
	width  int
	height int
}

func TestResultErr(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")

	tests := []struct {
		name     string
		result   variants.Result
		wantErrs []error
	}{
		{
			name: "no failures",
		},
		{
			name: "multiple failures",
			result: variants.Result{Failed: []variants.Failure{
				{Err: firstErr},
				{Err: secondErr},
			}},
			wantErrs: []error{firstErr, secondErr},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.result.Err()
			if len(tc.wantErrs) == 0 {
				if got != nil {
					t.Errorf("Result.Err() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("Result.Err() = nil, want an error")
			}
			for _, want := range tc.wantErrs {
				if !errors.Is(got, want) {
					t.Errorf("Result.Err() = %v, want error wrapping %v", got, want)
				}
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	standardSource := mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg"))

	tests := []struct {
		name          string
		source        paths.AbsPath
		widths        []int
		formats       []image.Format
		wantGenerated []image.Variant
	}{
		{
			name:    "generates supported formats in request order",
			source:  standardSource,
			widths:  []int{400, 800},
			formats: []image.Format{image.FormatJPEG, image.FormatAVIF},
			wantGenerated: []image.Variant{
				{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400, Height: 534},
				{Path: mustRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800, Height: 1067},
				{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400, Height: 534},
				{Path: mustRel(t, "w800.avif"), Format: image.FormatAVIF, Width: 800, Height: 1067},
			},
		},
		{
			name:    "does not enlarge images",
			source:  standardSource,
			widths:  []int{1600},
			formats: []image.Format{image.FormatJPEG, image.FormatAVIF},
			wantGenerated: []image.Variant{
				{Path: mustRel(t, "w1600.jpg"), Format: image.FormatJPEG, Width: 800, Height: 1067},
				{Path: mustRel(t, "w1600.avif"), Format: image.FormatAVIF, Width: 800, Height: 1067},
			},
		},
		{
			name:    "records auto-rotated dimensions",
			source:  mustAbs(t, filepath.Join("testdata", "image_800x600_orientation_8.jpg")),
			widths:  []int{400},
			formats: []image.Format{image.FormatJPEG, image.FormatAVIF},
			wantGenerated: []image.Variant{
				{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400, Height: 533},
				{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400, Height: 533},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := variants.Request{
				SourcePath: tc.source,
				OutputDir:  mustAbs(t, t.TempDir()),
				Widths:     tc.widths,
				Formats:    tc.formats,
			}
			got, err := variants.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("variants.Generate(%+v) returned error: %v", req, err)
			}
			if !slices.Equal(got.Generated, tc.wantGenerated) {
				t.Errorf("variants.Generate(%+v) generated variants mismatch\n got: %+v\nwant: %+v", req, got.Generated, tc.wantGenerated)
			}
			if len(got.Failed) != 0 {
				t.Errorf("variants.Generate(%+v) returned failed variants: %+v", req, got.Failed)
			}
			for _, variant := range got.Generated {
				outputPath, err := paths.JoinAbs(req.OutputDir, variant.Path)
				if err != nil {
					t.Fatalf("paths.JoinAbs(%q, %q) returned error: %v", req.OutputDir, variant.Path, err)
				}
				gotSize := readImageSize(t, outputPath)
				wantSize := imageSize{width: variant.Width, height: variant.Height}
				if gotSize != wantSize {
					t.Errorf("generated image %q size mismatch\n got: %+v\nwant: %+v", outputPath, gotSize, wantSize)
				}
			}
		})
	}
}

func TestGeneratePreservesExistingDestination(t *testing.T) {
	outputDir := mustAbs(t, t.TempDir())
	existingPath, err := paths.JoinAbs(outputDir, mustRel(t, "w400.jpg"))
	if err != nil {
		t.Fatalf("paths.JoinAbs(%q, %q) returned error: %v", outputDir, "w400.jpg", err)
	}
	wantContents := []byte("existing variant contents")
	if err := os.WriteFile(existingPath.String(), wantContents, 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) returned error: %v", existingPath, err)
	}

	req := variants.Request{
		SourcePath: mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg")),
		OutputDir:  outputDir,
		Widths:     []int{400, 800},
		Formats:    []image.Format{image.FormatJPEG},
	}
	got, err := variants.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("variants.Generate(%+v) returned request error: %v", req, err)
	}

	wantGenerated := []image.Variant{
		{Path: mustRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800, Height: 1067},
	}
	if !slices.Equal(got.Generated, wantGenerated) {
		t.Errorf("variants.Generate(%+v) generated variants mismatch\n got: %+v\nwant: %+v", req, got.Generated, wantGenerated)
	}
	if len(got.Failed) != 1 {
		t.Fatalf("variants.Generate(%+v) failed variant count = %d, want 1: %+v", req, len(got.Failed), got.Failed)
	}
	failed := got.Failed[0]
	if failed.Format != image.FormatJPEG || failed.Width != 400 {
		t.Errorf("variants.Generate(%+v) failed variant = %+v, want JPEG with width 400", req, failed)
	}
	if failed.Err == nil || !strings.Contains(failed.Err.Error(), "already exists") {
		t.Errorf("variants.Generate(%+v) failed error = %v, want message containing %q", req, failed.Err, "already exists")
	}

	gotContents, err := os.ReadFile(existingPath.String())
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned error: %v", existingPath, err)
	}
	if !bytes.Equal(gotContents, wantContents) {
		t.Errorf("existing destination contents = %q, want %q", gotContents, wantContents)
	}
}

func TestGenerateReturnsPartialResult(t *testing.T) {
	req := variants.Request{
		SourcePath: mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg")),
		OutputDir:  mustAbs(t, t.TempDir()),
		Widths:     []int{400},
		Formats:    []image.Format{image.FormatJPEG, image.FormatOther, image.FormatAVIF},
	}
	wantGenerated := []image.Variant{
		{Path: mustRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400, Height: 534},
		{Path: mustRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400, Height: 534},
	}
	wantProblem := "unsupported output format"

	got, err := variants.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("variants.Generate(%+v) returned request error: %v", req, err)
	}
	variantErr := got.Err()
	if variantErr == nil {
		t.Fatalf("variants.Generate(%+v) result error = nil, want error", req)
	}
	if !strings.Contains(variantErr.Error(), wantProblem) {
		t.Errorf("variants.Generate(%+v) result error = %v, want message containing %q", req, variantErr, wantProblem)
	}

	if !slices.Equal(got.Generated, wantGenerated) {
		t.Errorf("variants.Generate(%+v) generated variants mismatch\n got: %+v\nwant: %+v", req, got.Generated, wantGenerated)
	}

	failed := got.Failed
	if len(failed) != 1 {
		t.Fatalf("variants.Generate(%+v) failed variant count = %d, want 1: %+v", req, len(failed), failed)
	}
	if failed[0].Format != image.FormatOther || failed[0].Width != 400 {
		t.Errorf("variants.Generate(%+v) failed variant = %+v, want format %q and width %d", req, failed[0], image.FormatOther, 400)
	}
	if failed[0].Err == nil {
		t.Fatalf("variants.Generate(%+v) failed error is nil, want error containing %q", req, wantProblem)
	}
	if !strings.Contains(failed[0].Err.Error(), wantProblem) {
		t.Errorf("variants.Generate(%+v) failed error = %v, want message containing %q", req, failed[0].Err, wantProblem)
	}
}

func TestGenerateReportsVariantProblems(t *testing.T) {
	source := mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg"))

	tests := []struct {
		name        string
		request     func(*testing.T) variants.Request
		wantFormat  image.Format
		wantWidth   int
		wantProblem string
	}{
		{
			name: "empty output directory",
			request: func(t *testing.T) variants.Request {
				return variants.Request{
					SourcePath: source,
					Widths:     []int{400},
					Formats:    []image.Format{image.FormatJPEG},
				}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   400,
			wantProblem: "output directory path cannot be empty",
		},
		{
			name: "non-positive width",
			request: func(t *testing.T) variants.Request {
				return variants.Request{
					SourcePath: source,
					OutputDir:  mustAbs(t, t.TempDir()),
					Widths:     []int{0},
					Formats:    []image.Format{image.FormatJPEG},
				}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   0,
			wantProblem: "width must be positive",
		},
		{
			name: "source and output are the same file",
			request: func(t *testing.T) variants.Request {
				dir := t.TempDir()
				return variants.Request{
					SourcePath: mustAbs(t, filepath.Join(dir, "w400.jpg")),
					OutputDir:  mustAbs(t, dir),
					Widths:     []int{400},
					Formats:    []image.Format{image.FormatJPEG},
				}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   400,
			wantProblem: "source and output file paths cannot be the same",
		},
		{
			name: "unsupported output format",
			request: func(t *testing.T) variants.Request {
				return variants.Request{
					SourcePath: source,
					OutputDir:  mustAbs(t, t.TempDir()),
					Widths:     []int{400},
					Formats:    []image.Format{image.FormatOther},
				}
			},
			wantFormat:  image.FormatOther,
			wantWidth:   400,
			wantProblem: "unsupported output format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.request(t)
			got, err := variants.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("variants.Generate(%+v) returned request error: %v", req, err)
			}
			variantErr := got.Err()
			if variantErr == nil {
				t.Fatalf("variants.Generate(%+v) result error = nil, want error containing %q", req, tc.wantProblem)
			}
			if !strings.Contains(variantErr.Error(), tc.wantProblem) {
				t.Errorf("variants.Generate(%+v) result error = %v, want message containing %q", req, variantErr, tc.wantProblem)
			}

			if len(got.Generated) != 0 {
				t.Errorf("variants.Generate(%+v) generated variants: %+v, want none", req, got.Generated)
			}

			failed := got.Failed
			if len(failed) != 1 {
				t.Fatalf("variants.Generate(%+v) failed variant count = %d, want 1: %+v", req, len(failed), failed)
			}
			if failed[0].Format != tc.wantFormat || failed[0].Width != tc.wantWidth {
				t.Errorf("variants.Generate(%+v) failed variant = %+v, want format %q and width %d", req, failed[0], tc.wantFormat, tc.wantWidth)
			}
			if failed[0].Err == nil {
				t.Fatalf("variants.Generate(%+v) failed error is nil, want error containing %q", req, tc.wantProblem)
			}
			if !strings.Contains(failed[0].Err.Error(), tc.wantProblem) {
				t.Errorf("variants.Generate(%+v) failed error = %v, want message containing %q", req, failed[0].Err, tc.wantProblem)
			}
		})
	}
}

func TestGenerateClassifiesErrors(t *testing.T) {
	t.Run("empty source path is a request error", func(t *testing.T) {
		req := variants.Request{
			OutputDir: mustAbs(t, t.TempDir()),
			Widths:    []int{400},
			Formats:   []image.Format{image.FormatJPEG},
		}
		got, err := variants.Generate(context.Background(), req)
		wantProblem := "source file path cannot be empty"

		if err == nil {
			t.Fatalf("variants.Generate(%+v) returned nil request error, want error containing %q", req, wantProblem)
		}
		if !strings.Contains(err.Error(), wantProblem) {
			t.Errorf("variants.Generate(%+v) request error = %v, want message containing %q", req, err, wantProblem)
		}
		if len(got.Generated) != 0 || len(got.Failed) != 0 {
			t.Errorf("variants.Generate(%+v) result = %+v, want empty result", req, got)
		}
	})

	t.Run("vipsthumbnail command failure is a variant error", func(t *testing.T) {
		req := variants.Request{
			SourcePath: mustAbs(t, filepath.Join("testdata", "nonexistent.jpg")),
			OutputDir:  mustAbs(t, t.TempDir()),
			Widths:     []int{400},
			Formats:    []image.Format{image.FormatJPEG},
		}
		got, err := variants.Generate(context.Background(), req)

		if err != nil {
			t.Fatalf("variants.Generate(%+v) returned request error: %v", req, err)
		}
		variantErr := got.Err()
		if variantErr == nil {
			t.Fatalf("variants.Generate(%+v) result error = nil, want error wrapping *exec.ExitError", req)
		}
		if _, ok := errors.AsType[*exec.ExitError](variantErr); !ok {
			t.Errorf("variants.Generate(%+v) result error %v (%T), want error wrapping *exec.ExitError", req, variantErr, variantErr)
		}
		if !strings.Contains(variantErr.Error(), "image generation failed") {
			t.Errorf("variants.Generate(%+v) result error = %v, want message containing %q", req, variantErr, "image generation failed")
		}
		if len(got.Generated) != 0 {
			t.Errorf("variants.Generate(%+v) generated variants: %+v, want none", req, got.Generated)
		}
		if len(got.Failed) != 1 || got.Failed[0].Format != image.FormatJPEG || got.Failed[0].Width != 400 {
			t.Errorf("variants.Generate(%+v) failed variants = %+v, want one JPEG failure with width 400", req, got.Failed)
		}
	})
}

func TestGenerateCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := variants.Request{
		SourcePath: mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg")),
		OutputDir:  mustAbs(t, t.TempDir()),
		Widths:     []int{400},
		Formats:    []image.Format{image.FormatJPEG},
	}
	got, err := variants.Generate(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("variants.Generate(%+v) error = %v, want context.Canceled", req, err)
	}
	if len(got.Generated) != 0 || len(got.Failed) != 0 {
		t.Errorf("variants.Generate(%+v) result = %+v, want empty result", req, got)
	}
}

func readImageSize(t *testing.T, path paths.AbsPath) imageSize {
	t.Helper()

	// -s3 causes exiftool to output the values of the fields only
	out, err := exec.Command("exiftool", "-ImageWidth", "-ImageHeight", "-s3", path.String()).Output()
	if err != nil {
		t.Fatalf("reading image size for %q: %v", path, err)
	}

	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		t.Fatalf("reading image size for %q returned %q, want width and height", path, out)
	}

	width, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("parsing image width for %q: %v", path, err)
	}
	height, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("parsing image height for %q: %v", path, err)
	}

	return imageSize{width: width, height: height}
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

func mustRel(t *testing.T, path string) paths.RelPath {
	t.Helper()

	p, err := paths.NewRelPath(path)
	if err != nil {
		t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", path, err)
	}

	return p
}
