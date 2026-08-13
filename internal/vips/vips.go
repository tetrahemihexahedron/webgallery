package vips

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Spec struct {
	OutPath string
	Width   int
}

type Result struct {
	Variants []VariantResult
}

type VariantResult struct {
	Spec      Spec
	Generated bool
	Err       error
}

func (r Result) Generated() []Spec {
	var generated []Spec
	for _, variant := range r.Variants {
		if variant.Generated {
			generated = append(generated, variant.Spec)
		}
	}
	return generated
}

func (r Result) Failed() []VariantResult {
	var failed []VariantResult
	for _, variant := range r.Variants {
		if variant.Err != nil {
			failed = append(failed, variant)
		}
	}
	return failed
}

func (r Result) Err() error {
	var errs []error
	for _, result := range r.Failed() {
		errs = append(errs, result.Err)
	}
	return errors.Join(errs...)
}

type Vipsthumbnail struct{}

func (v *Vipsthumbnail) Generate(source string, specs []Spec) (Result, error) {
	if source == "" {
		return Result{}, errors.New("source file path cannot be empty")
	}

	result := Result{
		Variants: make([]VariantResult, 0, len(specs)),
	}

	for _, spec := range specs {
		variantResult := generateVariant(source, spec)
		result.Variants = append(result.Variants, variantResult)
	}
	return result, result.Err()
}

func generateVariant(source string, spec Spec) VariantResult {
	result := VariantResult{
		Spec: spec,
	}

	if err := validateSpec(spec); err != nil {
		result.Err = wrapError(err, spec)
		return result
	}
	if filepath.Clean(source) == filepath.Clean(spec.OutPath) {
		result.Err = wrapError(errors.New("source and output file paths cannot be the same"), spec)
		return result
	}

	options, err := determineEncoderOptions(spec.OutPath)
	if err != nil {
		result.Err = wrapError(err, spec)
		return result
	}

	// appending '>' tells libvips to only shrink; if the image is already
	// smaller than the requested size, the size won't change
	width := strconv.Itoa(spec.Width) + "x>"
	path := spec.OutPath + options

	cmd := exec.Command("vipsthumbnail", source, "--size", width, "--path", path)

	out, err := cmd.CombinedOutput()

	if err != nil {
		result.Err = wrapError(fmt.Errorf("image generation failed: %s; %w", out, err), spec)
		return result
	}
	// out is expected to be empty when image generation was successful
	if len(out) != 0 {
		result.Err = wrapError(fmt.Errorf("unexpected output from image generation: %s", out), spec)
		return result
	}

	result.Generated = true
	return result
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
