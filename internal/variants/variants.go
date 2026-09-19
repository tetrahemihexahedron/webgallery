package variants

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/paths"
)

// Request describes the source and output variants to generate.
type Request struct {
	SourcePath paths.AbsPath
	OutputDir  paths.AbsPath
	Widths     []int
	Formats    []image.Format
}

// RequestResult reports which requested variants succeeded or failed.
type RequestResult struct {
	Generated []image.Variant
	Failed    []RequestFailure
}

// RequestFailure reports why one requested variant could not be generated.
type RequestFailure struct {
	Format image.Format
	Width  int
	Err    error
}

// Err returns the errors reported for individual requested variants.
func (r RequestResult) Err() error {
	var errs []error
	for _, failure := range r.Failed {
		errs = append(errs, failure.Err)
	}
	return errors.Join(errs...)
}

type plannedVariant struct {
	variant        image.Variant
	outputPath     paths.AbsPath
	encoderOptions string
}

// GenerateRequest attempts every requested format and width combination unless
// the request is invalid. It returns request-level validation errors directly
// with an empty RequestResult. Errors from individual attempts are recorded in
// RequestResult.Failed and available through RequestResult.Err; they do not
// make GenerateRequest return an error.
func GenerateRequest(req Request) (RequestResult, error) {
	if req.SourcePath.String() == "" {
		return RequestResult{}, errors.New("source file path cannot be empty")
	}

	result := RequestResult{
		Generated: make([]image.Variant, 0, len(req.Widths)*len(req.Formats)),
	}

	for _, format := range req.Formats {
		for _, width := range req.Widths {
			planned, err := planVariant(req.OutputDir, format, width)
			if err == nil {
				err = generatePlannedVariant(req.SourcePath, planned)
			}
			if err != nil {
				result.Failed = append(result.Failed, RequestFailure{
					Format: format,
					Width:  width,
					Err:    wrapRequestError(err, format, width),
				})
				continue
			}
			result.Generated = append(result.Generated, planned.variant)
		}
	}

	return result, nil
}

func planVariant(outputDir paths.AbsPath, format image.Format, width int) (plannedVariant, error) {
	if outputDir.String() == "" {
		return plannedVariant{}, errors.New("output directory path cannot be empty")
	}
	if width <= 0 {
		return plannedVariant{}, errors.New("width must be positive")
	}

	extension, encoderOptions, err := formatSettings(format)
	if err != nil {
		return plannedVariant{}, err
	}

	path, err := paths.NewRelPath("w" + strconv.Itoa(width) + extension)
	if err != nil {
		return plannedVariant{}, err
	}
	outputPath, err := paths.JoinAbs(outputDir, path)
	if err != nil {
		return plannedVariant{}, err
	}

	return plannedVariant{
		variant: image.Variant{
			Path:   path,
			Format: format,
			Width:  width,
		},
		outputPath:     outputPath,
		encoderOptions: encoderOptions,
	}, nil
}

func generatePlannedVariant(source paths.AbsPath, planned plannedVariant) error {
	if source == planned.outputPath {
		return errors.New("source and output file paths cannot be the same")
	}

	// appending '>' tells libvips to only shrink; if the image is already
	// smaller than the requested size, the size won't change
	sizeArg := strconv.Itoa(planned.variant.Width) + "x>"
	outputArg := planned.outputPath.String() + planned.encoderOptions

	cmd := exec.Command("vipsthumbnail", source.String(), "--size", sizeArg, "--output", outputArg)

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("image generation failed: %s; %w", cmdOutput, err)
	}
	// cmdOutput is expected to be empty when image generation was successful
	if len(cmdOutput) != 0 {
		return fmt.Errorf("unexpected output from image generation: %s", cmdOutput)
	}

	return nil
}

func wrapRequestError(err error, format image.Format, width int) error {
	return fmt.Errorf("generating %s variant with width %d: %w", format, width, err)
}

func formatSettings(format image.Format) (extension, encoderOptions string, err error) {
	switch format {
	case image.FormatJPEG:
		return ".jpg", "[Q=75,keep=none]", nil
	case image.FormatAVIF:
		return ".avif", "[Q=75,effort=6,keep=none]", nil
	default:
		return "", "", fmt.Errorf("unsupported output format %q", format)
	}
}
