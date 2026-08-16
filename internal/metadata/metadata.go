package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"tetrahemihexahedron/webimage/internal/image"
)

type Result struct {
	Metadata     []image.Metadata
	FileProblems []Problem
}

type Problem struct {
	FileName string
	Message  string
}

type Exiftool struct{}

func (e *Exiftool) Read(path string) (Result, error) {
	output, err := fetchExiftoolOutput(path)
	if err != nil {
		return Result{}, err
	}

	result := processOutput(output)
	return result, nil
}

type exiftoolOutput struct {
	FileName    string `json:"FileName"`
	FileType    string `json:"FileType"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Width       int    `json:"ImageWidth"`
	Height      int    `json:"ImageHeight"`
	Error       string `json:"Error"`
}

func fetchExiftoolOutput(path string) ([]exiftoolOutput, error) {
	cmd := exec.Command("exiftool", "-json", path)

	rawOutput, commandErr := cmd.Output()

	if len(rawOutput) == 0 {
		if commandErr == nil {
			return nil, fmt.Errorf("running exiftool on %q returned no output and no error", path)
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](commandErr); ok {
			return nil, fmt.Errorf("running exiftool on %q: %w %s", path, commandErr, exitErr.Stderr)
		}
		return nil, fmt.Errorf("running exiftool on %q: %w", path, commandErr)
	}

	// if there was output to stdout, then commandErr is probably not interesting
	// because exiftool reports the reasons for errors in stdout
	var output []exiftoolOutput
	if err := json.Unmarshal(rawOutput, &output); err != nil {
		return nil, fmt.Errorf("unmarshalling exiftool output for %q: %w", path, err)
	}
	return output, nil
}

func processOutput(output []exiftoolOutput) Result {
	metadata := make([]image.Metadata, 0, len(output))
	var problems []Problem

	for _, out := range output {
		if out.Error != "" {
			problems = append(problems, Problem{
				FileName: out.FileName,
				Message:  fmt.Sprintf("reported by exiftool: %s", out.Error),
			})
			continue
		}
		missing := missingMetadata(out)
		if len(missing) != 0 {
			problems = append(problems, Problem{
				FileName: out.FileName,
				Message:  fmt.Sprintf("missing required metadata: %v", missing),
			})
			continue
		}

		metadata = append(metadata, image.Metadata{
			FileName:    out.FileName,
			Format:      out.FileType,
			Title:       out.Title,
			Description: out.Description,
			Width:       out.Width,
			Height:      out.Height,
		})
	}
	return Result{
		Metadata:     metadata,
		FileProblems: problems,
	}
}

func missingMetadata(out exiftoolOutput) []string {
	required := map[string]bool{
		"FileName": out.FileName == "",
		"FileType": out.FileType == "",
		"Width":    out.Width == 0,
		"Height":   out.Height == 0,
	}
	var missing []string

	for field, isMissing := range required {
		if isMissing {
			missing = append(missing, field)
		}
	}
	return missing
}
