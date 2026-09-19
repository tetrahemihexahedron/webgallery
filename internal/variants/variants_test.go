package variants_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/paths"
	"tetrahemihexahedron/webimage/internal/variants"
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

func TestVipsthumbnailGenerate(t *testing.T) {
	dir := t.TempDir()
	source := mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg"))

	tests := []struct {
		name      string
		specs     []variants.Spec
		wantSizes []imageSize
	}{
		{
			name: "generates supported formats",
			specs: []variants.Spec{
				{OutPath: mustAbs(t, filepath.Join(dir, "w400.jpg")), Width: 400},
				{OutPath: mustAbs(t, filepath.Join(dir, "w400.jpeg")), Width: 400},
				{OutPath: mustAbs(t, filepath.Join(dir, "w400.avif")), Width: 400},
			},
			wantSizes: []imageSize{
				{width: 400, height: 534},
				{width: 400, height: 534},
				{width: 400, height: 534},
			},
		},
		{
			name: "does not enlarge images",
			specs: []variants.Spec{
				{OutPath: mustAbs(t, filepath.Join(dir, "w1600.jpg")), Width: 1600},
			},
			wantSizes: []imageSize{
				{width: 800, height: 1067},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.specs) != len(tc.wantSizes) {
				t.Fatalf("test case has %d specs and %d wantSizes, want equal lengths", len(tc.specs), len(tc.wantSizes))
			}

			got, err := (&variants.Vipsthumbnail{}).Generate(source, tc.specs)
			if err != nil {
				t.Fatalf("Vipsthumbnail.Generate(%q, %+v) returned error: %v", source, tc.specs, err)
			}
			if !slices.Equal(got.Generated, tc.specs) {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) generated specs mismatch\n got: %+v\nwant: %+v", source, tc.specs, got.Generated, tc.specs)
			}
			if len(got.Failed) != 0 {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) returned failed variants: %+v", source, tc.specs, got.Failed)
			}
			for i, spec := range tc.specs {
				gotSize := readImageSize(t, spec.OutPath)
				wantSize := tc.wantSizes[i]
				if gotSize != wantSize {
					t.Errorf("generated image %q size mismatch\n got: %+v\nwant: %+v", spec.OutPath, gotSize, wantSize)
				}
			}
		})
	}
}

func TestVipsthumbnailGenerateReturnsPartialResult(t *testing.T) {
	dir := t.TempDir()
	source := mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg"))

	validJPG := variants.Spec{
		OutPath: mustAbs(t, filepath.Join(dir, "w400.jpg")),
		Width:   400,
	}
	invalid := variants.Spec{
		OutPath: mustAbs(t, filepath.Join(dir, "w400.webp")),
		Width:   400,
	}
	validAVIF := variants.Spec{
		OutPath: mustAbs(t, filepath.Join(dir, "w400.avif")),
		Width:   400,
	}
	specs := []variants.Spec{validJPG, invalid, validAVIF}
	wantGenerated := []variants.Spec{validJPG, validAVIF}
	wantProblem := "unsupported output file extension"

	got, err := (&variants.Vipsthumbnail{}).Generate(source, specs)
	if err == nil {
		t.Fatalf("Vipsthumbnail.Generate(%q, %+v) returned nil error, want error", source, specs)
	}
	if !strings.Contains(err.Error(), wantProblem) {
		t.Errorf("Vipsthumbnail.Generate(%q, %+v) error = %v, want message containing %q", source, specs, err, wantProblem)
	}

	if !slices.Equal(got.Generated, wantGenerated) {
		t.Errorf("Vipsthumbnail.Generate(%q, %+v) generated specs mismatch\n got: %+v\nwant: %+v", source, specs, got.Generated, wantGenerated)
	}

	failed := got.Failed
	if len(failed) != 1 {
		t.Fatalf("Vipsthumbnail.Generate(%q, %+v) failed variant count = %d, want 1: %+v", source, specs, len(failed), failed)
	}
	if failed[0].Spec != invalid {
		t.Errorf("Vipsthumbnail.Generate(%q, %+v) failed spec mismatch\n got: %+v\nwant: %+v", source, specs, failed[0].Spec, invalid)
	}
	if failed[0].Err == nil {
		t.Fatalf("Vipsthumbnail.Generate(%q, %+v) failed error is nil, want error containing %q", source, specs, wantProblem)
	}
	if !strings.Contains(failed[0].Err.Error(), wantProblem) {
		t.Errorf("Vipsthumbnail.Generate(%q, %+v) failed error = %v, want message containing %q", source, specs, failed[0].Err, wantProblem)
	}
}

func TestVipsthumbnailGenerateReportsSpecProblems(t *testing.T) {
	dir := t.TempDir()
	source := mustAbs(t, filepath.Join("testdata", "image_800x1067.jpg"))

	tests := []struct {
		name        string
		spec        variants.Spec
		wantProblem string
	}{
		{
			name:        "empty output path",
			spec:        variants.Spec{Width: 400},
			wantProblem: "output file path cannot be empty",
		},
		{
			name:        "non-positive width",
			spec:        variants.Spec{OutPath: mustAbs(t, filepath.Join(dir, "w0.jpg")), Width: 0},
			wantProblem: "width must be positive",
		},
		{
			name:        "source and output are the same file",
			spec:        variants.Spec{OutPath: source, Width: 400},
			wantProblem: "source and output file paths cannot be the same",
		},
		{
			name:        "unsupported output extension",
			spec:        variants.Spec{OutPath: mustAbs(t, filepath.Join(dir, "w400.webp")), Width: 400},
			wantProblem: "unsupported output file extension",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.spec
			specs := []variants.Spec{spec}
			got, err := (&variants.Vipsthumbnail{}).Generate(source, specs)

			if err == nil {
				t.Fatalf("Vipsthumbnail.Generate(%q, %+v) returned nil error, want error containing %q", source, specs, tc.wantProblem)
			}
			if !strings.Contains(err.Error(), tc.wantProblem) {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) error = %v, want message containing %q", source, specs, err, tc.wantProblem)
			}

			if len(got.Generated) != 0 {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) generated specs: %+v, want none", source, specs, got.Generated)
			}

			failed := got.Failed
			if len(failed) != 1 {
				t.Fatalf("Vipsthumbnail.Generate(%q, %+v) failed variant count = %d, want 1: %+v", source, specs, len(failed), failed)
			}
			if failed[0].Spec != spec {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) failed spec mismatch\n got: %+v\nwant: %+v", source, specs, failed[0].Spec, spec)
			}
			if failed[0].Err == nil {
				t.Fatalf("Vipsthumbnail.Generate(%q, %+v) failed error is nil, want error containing %q", source, specs, tc.wantProblem)
			}
			if !strings.Contains(failed[0].Err.Error(), tc.wantProblem) {
				t.Errorf("Vipsthumbnail.Generate(%q, %+v) failed error = %v, want message containing %q", source, specs, failed[0].Err, tc.wantProblem)
			}
		})
	}
}

func TestVipsthumbnailGenerateReturnsError(t *testing.T) {
	t.Run("empty source path", func(t *testing.T) {
		var source paths.AbsPath
		specs := []variants.Spec{{OutPath: mustAbs(t, filepath.Join(t.TempDir(), "w400.jpg")), Width: 400}}
		got, err := (&variants.Vipsthumbnail{}).Generate(source, specs)
		wantProblem := "source file path cannot be empty"

		if err == nil {
			t.Fatalf("Vipsthumbnail.Generate(%q, %+v) returned nil error, want error containing %q", source, specs, wantProblem)
		}
		if !strings.Contains(err.Error(), wantProblem) {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) error = %v, want message containing %q", source, specs, err, wantProblem)
		}
		if len(got.Generated) != 0 || len(got.Failed) != 0 {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) result = %+v, want no generated or failed variants", source, specs, got)
		}
	})

	t.Run("vipsthumbnail command fails for missing source", func(t *testing.T) {
		source := mustAbs(t, filepath.Join("testdata", "nonexistent.jpg"))
		spec := variants.Spec{OutPath: mustAbs(t, filepath.Join(t.TempDir(), "w400.jpg")), Width: 400}
		specs := []variants.Spec{spec}
		got, err := (&variants.Vipsthumbnail{}).Generate(source, specs)

		if err == nil {
			t.Fatalf("Vipsthumbnail.Generate(%q, %+v) returned nil error, want error wrapping *exec.ExitError", source, specs)
		}
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) returned error %v (%T), want error wrapping *exec.ExitError", source, specs, err, err)
		}
		if !strings.Contains(err.Error(), "image generation failed") {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) error = %v, want message containing %q", source, specs, err, "image generation failed")
		}
		if len(got.Generated) != 0 {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) generated specs: %+v, want none", source, specs, got.Generated)
		}
		if len(got.Failed) != 1 || got.Failed[0].Spec != spec {
			t.Errorf("Vipsthumbnail.Generate(%q, %+v) failed variants = %+v, want one failure for %+v", source, specs, got.Failed, spec)
		}
	})
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
