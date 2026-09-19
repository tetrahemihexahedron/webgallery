package variants_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/paths"
	"tetrahemihexahedron/webimage/internal/variants"
)

func TestRequestResultErr(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")

	tests := []struct {
		name     string
		result   variants.RequestResult
		wantErrs []error
	}{
		{
			name: "no failures",
		},
		{
			name: "multiple failures",
			result: variants.RequestResult{Failed: []variants.RequestFailure{
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
					t.Errorf("RequestResult.Err() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("RequestResult.Err() = nil, want an error")
			}
			for _, want := range tc.wantErrs {
				if !errors.Is(got, want) {
					t.Errorf("RequestResult.Err() = %v, want error wrapping %v", got, want)
				}
			}
		})
	}
}

func TestGenerateRequest(t *testing.T) {
	req := variants.Request{
		SourcePath: mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg")),
		OutputDir:  mustAbs(t, t.TempDir()),
		Widths:     []int{400, 800},
		Formats:    []image.Format{image.FormatJPEG, image.FormatAVIF},
	}
	wantGenerated := []image.Variant{
		{Path: mustRequestRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400},
		{Path: mustRequestRel(t, "w800.jpg"), Format: image.FormatJPEG, Width: 800},
		{Path: mustRequestRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400},
		{Path: mustRequestRel(t, "w800.avif"), Format: image.FormatAVIF, Width: 800},
	}
	wantSizes := []imageSize{
		{width: 400, height: 534},
		{width: 800, height: 1067},
		{width: 400, height: 534},
		{width: 800, height: 1067},
	}

	got, err := variants.GenerateRequest(req)
	if err != nil {
		t.Fatalf("variants.GenerateRequest(%+v) returned error: %v", req, err)
	}
	if !slices.Equal(got.Generated, wantGenerated) {
		t.Errorf("variants.GenerateRequest(%+v) generated variants mismatch\n got: %+v\nwant: %+v", req, got.Generated, wantGenerated)
	}
	if len(got.Failed) != 0 {
		t.Errorf("variants.GenerateRequest(%+v) returned failed variants: %+v", req, got.Failed)
	}
	for i, variant := range got.Generated {
		outputPath, err := paths.JoinAbs(req.OutputDir, variant.Path)
		if err != nil {
			t.Fatalf("paths.JoinAbs(%q, %q) returned error: %v", req.OutputDir, variant.Path, err)
		}
		gotSize := readImageSize(t, outputPath)
		if gotSize != wantSizes[i] {
			t.Errorf("generated image %q size mismatch\n got: %+v\nwant: %+v", outputPath, gotSize, wantSizes[i])
		}
	}
}

func TestGenerateRequestReturnsPartialResult(t *testing.T) {
	req := variants.Request{
		SourcePath: mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg")),
		OutputDir:  mustAbs(t, t.TempDir()),
		Widths:     []int{400},
		Formats:    []image.Format{image.FormatJPEG, image.FormatOther, image.FormatAVIF},
	}
	wantGenerated := []image.Variant{
		{Path: mustRequestRel(t, "w400.jpg"), Format: image.FormatJPEG, Width: 400},
		{Path: mustRequestRel(t, "w400.avif"), Format: image.FormatAVIF, Width: 400},
	}
	wantProblem := "unsupported output format"

	got, err := variants.GenerateRequest(req)
	if err != nil {
		t.Fatalf("variants.GenerateRequest(%+v) returned request error: %v", req, err)
	}
	if !slices.Equal(got.Generated, wantGenerated) {
		t.Errorf("variants.GenerateRequest(%+v) generated variants mismatch\n got: %+v\nwant: %+v", req, got.Generated, wantGenerated)
	}
	if len(got.Failed) != 1 {
		t.Fatalf("variants.GenerateRequest(%+v) failed variant count = %d, want 1: %+v", req, len(got.Failed), got.Failed)
	}
	failure := got.Failed[0]
	if failure.Format != image.FormatOther || failure.Width != 400 {
		t.Errorf("variants.GenerateRequest(%+v) failed variant = %+v, want format %q and width %d", req, failure, image.FormatOther, 400)
	}
	if failure.Err == nil || !strings.Contains(failure.Err.Error(), wantProblem) {
		t.Errorf("variants.GenerateRequest(%+v) failed error = %v, want message containing %q", req, failure.Err, wantProblem)
	}
}

func TestGenerateRequestReportsVariantProblems(t *testing.T) {
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
				return variants.Request{SourcePath: source, Widths: []int{400}, Formats: []image.Format{image.FormatJPEG}}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   400,
			wantProblem: "output directory path cannot be empty",
		},
		{
			name: "non-positive width",
			request: func(t *testing.T) variants.Request {
				return variants.Request{SourcePath: source, OutputDir: mustAbs(t, t.TempDir()), Widths: []int{0}, Formats: []image.Format{image.FormatJPEG}}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   0,
			wantProblem: "width must be positive",
		},
		{
			name: "source and output are the same file",
			request: func(t *testing.T) variants.Request {
				dir := t.TempDir()
				return variants.Request{SourcePath: mustAbs(t, filepath.Join(dir, "w400.jpg")), OutputDir: mustAbs(t, dir), Widths: []int{400}, Formats: []image.Format{image.FormatJPEG}}
			},
			wantFormat:  image.FormatJPEG,
			wantWidth:   400,
			wantProblem: "source and output file paths cannot be the same",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.request(t)
			got, err := variants.GenerateRequest(req)
			if err != nil {
				t.Fatalf("variants.GenerateRequest(%+v) returned request error: %v", req, err)
			}
			if len(got.Generated) != 0 || len(got.Failed) != 1 {
				t.Fatalf("variants.GenerateRequest(%+v) result = %+v, want one failure", req, got)
			}
			failure := got.Failed[0]
			if failure.Format != tc.wantFormat || failure.Width != tc.wantWidth {
				t.Errorf("variants.GenerateRequest(%+v) failed variant = %+v, want format %q and width %d", req, failure, tc.wantFormat, tc.wantWidth)
			}
			if failure.Err == nil || !strings.Contains(failure.Err.Error(), tc.wantProblem) {
				t.Errorf("variants.GenerateRequest(%+v) failed error = %v, want message containing %q", req, failure.Err, tc.wantProblem)
			}
		})
	}
}

func TestGenerateRequestClassifiesErrors(t *testing.T) {
	t.Run("empty source path is a request error", func(t *testing.T) {
		req := variants.Request{OutputDir: mustAbs(t, t.TempDir()), Widths: []int{400}, Formats: []image.Format{image.FormatJPEG}}
		got, err := variants.GenerateRequest(req)
		if err == nil {
			t.Fatalf("variants.GenerateRequest(%+v) returned nil request error, want error", req)
		}
		if len(got.Generated) != 0 || len(got.Failed) != 0 {
			t.Errorf("variants.GenerateRequest(%+v) result = %+v, want empty result", req, got)
		}
	})

	t.Run("vipsthumbnail command failure is a variant error", func(t *testing.T) {
		req := variants.Request{
			SourcePath: mustAbs(t, filepath.Join("testdata", "nonexistent.jpg")),
			OutputDir:  mustAbs(t, t.TempDir()),
			Widths:     []int{400},
			Formats:    []image.Format{image.FormatJPEG},
		}
		got, err := variants.GenerateRequest(req)
		if err != nil {
			t.Fatalf("variants.GenerateRequest(%+v) returned request error: %v", req, err)
		}
		variantErr := got.Err()
		if variantErr == nil {
			t.Fatalf("variants.GenerateRequest(%+v) result error = nil, want error wrapping *exec.ExitError", req)
		}
		if _, ok := errors.AsType[*exec.ExitError](variantErr); !ok {
			t.Errorf("variants.GenerateRequest(%+v) result error %v (%T), want error wrapping *exec.ExitError", req, variantErr, variantErr)
		}
	})
}

func mustRequestRel(t *testing.T, path string) paths.RelPath {
	t.Helper()

	p, err := paths.NewRelPath(path)
	if err != nil {
		t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", path, err)
	}
	return p
}
