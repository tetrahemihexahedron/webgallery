package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
	FileName         string `json:"FileName"`
	FileType         string `json:"FileType"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
	DateTimeOriginal string `json:"DateTimeOriginal"`
	Width            int    `json:"ImageWidth"`
	Height           int    `json:"ImageHeight"`
	Error            string `json:"Error"`
}

const dateTimeOriginalLayout = "2006:01:02 15:04:05"

func parseDateTimeOriginal(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("DateTimeOriginal is empty")
	}

	capturedAt, err := time.Parse(dateTimeOriginalLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing DateTimeOriginal %q: %w", s, err)
	}
	return capturedAt, nil
}

func fetchExiftoolOutput(path string) ([]exiftoolOutput, error) {
	cmd := exec.Command("exiftool", "-json", path)

	rawOutput, commandErr := cmd.Output()

	if len(rawOutput) == 0 {
		if commandErr == nil {
			// This is normal for an empty directory
			empty, err := isEmptyDir(path)
			if err != nil {
				return nil, fmt.Errorf("checking whether %q is an empty directory: %w", path, err)
			}
			if empty {
				return nil, nil
			}
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

func isEmptyDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
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

		capturedAt := ""
		if strings.TrimSpace(out.DateTimeOriginal) != "" {
			parsedCapturedAt, err := parseDateTimeOriginal(out.DateTimeOriginal)
			if err != nil {
				problems = append(problems, Problem{
					FileName: out.FileName,
					Message:  fmt.Sprintf("invalid DateTimeOriginal %q: %v", out.DateTimeOriginal, err),
				})
				continue
			}
			capturedAt = image.FormatCapturedAt(parsedCapturedAt)
		}

		metadata = append(metadata, image.Metadata{
			FileName:    out.FileName,
			Format:      out.FileType,
			Title:       out.Title,
			Description: out.Description,
			CapturedAt:  capturedAt,
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
	var missing []string

	if out.FileName == "" {
		missing = append(missing, "FileName")
	}
	if out.FileType == "" {
		missing = append(missing, "FileType")
	}
	if out.Width == 0 {
		missing = append(missing, "Width")
	}
	if out.Height == 0 {
		missing = append(missing, "Height")
	}

	return missing
}
