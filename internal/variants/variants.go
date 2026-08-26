package variants

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Spec describes one output image variant to generate.
type Spec struct {
	OutPath string
	Width   int
}

// Result reports which variants were generated and which failed.
type Result struct {
	Generated []Spec
	Failed    []Failure
}

// Failure reports why one variant could not be generated.
type Failure struct {
	Spec Spec
	Err  error
}

func (r Result) Err() error {
	var errs []error
	for _, failure := range r.Failed {
		errs = append(errs, failure.Err)
	}
	return errors.Join(errs...)
}

type Vipsthumbnail struct{}

func (v *Vipsthumbnail) Generate(source string, specs []Spec) (Result, error) {
	if source == "" {
		return Result{}, errors.New("source file path cannot be empty")
	}

	result := Result{
		Generated: make([]Spec, 0, len(specs)),
	}

	for _, spec := range specs {
		if err := generateVariant(source, spec); err != nil {
			result.Failed = append(result.Failed, Failure{Spec: spec, Err: err})
			continue
		}
		result.Generated = append(result.Generated, spec)
	}
	return result, result.Err()
}

func generateVariant(source string, spec Spec) error {
	if err := validateSpec(spec); err != nil {
		return wrapError(err, spec)
	}
	if filepath.Clean(source) == filepath.Clean(spec.OutPath) {
		return wrapError(errors.New("source and output file paths cannot be the same"), spec)
	}

	options, err := determineEncoderOptions(spec.OutPath)
	if err != nil {
		return wrapError(err, spec)
	}

	absoluteOutPath, err := filepath.Abs(spec.OutPath)
	if err != nil {
		return wrapError(fmt.Errorf("unable to form absolute output path: %w", err), spec)
	}

	// appending '>' tells libvips to only shrink; if the image is already
	// smaller than the requested size, the size won't change
	width := strconv.Itoa(spec.Width) + "x>"
	path := absoluteOutPath + options

	cmd := exec.Command("vipsthumbnail", source, "--size", width, "--output", path)

	out, err := cmd.CombinedOutput()

	if err != nil {
		return wrapError(fmt.Errorf("image generation failed: %s; %w", out, err), spec)
	}
	// out is expected to be empty when image generation was successful
	if len(out) != 0 {
		return wrapError(fmt.Errorf("unexpected output from image generation: %s", out), spec)
	}

	return nil
}

func wrapError(err error, spec Spec) error {
	return fmt.Errorf(
		"generating %q with width %d: %w",
		spec.OutPath,
		spec.Width,
		err,
	)
}

func validateSpec(spec Spec) error {
	if spec.OutPath == "" {
		return errors.New("output file path cannot be empty")
	}

	if spec.Width <= 0 {
		return errors.New("width must be positive")
	}

	return nil
}

func determineEncoderOptions(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpeg", ".jpg":
		return "[Q=75,keep=none]", nil
	case ".avif":
		return "[Q=75,effort=6,keep=none]", nil
	default:
		return "", fmt.Errorf(
			"unsupported output file extension %q",
			filepath.Ext(path),
		)
	}
}
