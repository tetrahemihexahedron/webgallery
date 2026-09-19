package variants

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tetrahemihexahedron/webimage/internal/paths"
)

// Spec describes one output image variant to generate.
type Spec struct {
	OutPath paths.AbsPath
	Width   int
}

// Result reports which individual variant attempts succeeded or failed.
// Generate reports request-level failures through its separate error return.
type Result struct {
	Generated []Spec
	Failed    []Failure
}

// Failure reports why one variant could not be generated.
type Failure struct {
	Spec Spec
	Err  error
}

// Err returns the errors reported for individual variant attempts.
func (r Result) Err() error {
	var errs []error
	for _, failure := range r.Failed {
		errs = append(errs, failure.Err)
	}
	return errors.Join(errs...)
}

type Vipsthumbnail struct{}

// Generate attempts every requested variant unless the request is invalid.
// It returns request-level validation errors directly with an empty Result.
// Errors from individual attempts are recorded in Result.Failed and available
// through Result.Err; they do not make Generate return an error.
func (v *Vipsthumbnail) Generate(source paths.AbsPath, specs []Spec) (Result, error) {
	if source.String() == "" {
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
	return result, nil
}

func generateVariant(source paths.AbsPath, spec Spec) error {
	if err := validateSpec(spec); err != nil {
		return wrapError(err, spec)
	}
	if source == spec.OutPath {
		return wrapError(errors.New("source and output file paths cannot be the same"), spec)
	}

	options, err := determineEncoderOptions(spec.OutPath)
	if err != nil {
		return wrapError(err, spec)
	}

	// appending '>' tells libvips to only shrink; if the image is already
	// smaller than the requested size, the size won't change
	sizeArg := strconv.Itoa(spec.Width) + "x>"
	outputArg := spec.OutPath.String() + options

	cmd := exec.Command("vipsthumbnail", source.String(), "--size", sizeArg, "--output", outputArg)

	cmdOutput, err := cmd.CombinedOutput()

	if err != nil {
		return wrapError(fmt.Errorf("image generation failed: %s; %w", cmdOutput, err), spec)
	}
	// cmdOutput is expected to be empty when image generation was successful
	if len(cmdOutput) != 0 {
		return wrapError(fmt.Errorf("unexpected output from image generation: %s", cmdOutput), spec)
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
	if spec.OutPath.String() == "" {
		return errors.New("output file path cannot be empty")
	}

	if spec.Width <= 0 {
		return errors.New("width must be positive")
	}

	return nil
}

func determineEncoderOptions(path paths.AbsPath) (string, error) {
	ext := filepath.Ext(path.String())
	switch strings.ToLower(ext) {
	case ".jpeg", ".jpg":
		return "[Q=75,keep=none]", nil
	case ".avif":
		return "[Q=75,effort=6,keep=none]", nil
	default:
		return "", fmt.Errorf("unsupported output file extension %q", ext)
	}
}
