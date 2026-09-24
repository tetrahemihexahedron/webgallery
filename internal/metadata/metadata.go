package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"tetrahemihexahedron/webgallery/internal/image"
	"tetrahemihexahedron/webgallery/internal/paths"
)

// File contains metadata extracted from one file.
// Format, Title, and Description have surrounding whitespace removed. FileName
// is not whitespace-normalized because it identifies a path on the filesystem.
type File struct {
	FileName    paths.RelPath
	Format      string
	Title       string
	Description string
	// CapturedAt is empty or exactly the value produced by image.FormatCapturedAt.
	CapturedAt string
	Width      int
	Height     int
}

// Result contains metadata records and per-file problems from a read.
// Metadata and FileProblems are sorted independently by filename in ascending
// lexical order; entries with equal filenames retain their original order.
type Result struct {
	Metadata     []File
	FileProblems []Problem
}

type Problem struct {
	FileName string
	Message  string
}

// Read extracts metadata from a file or directory using exiftool.
func Read(ctx context.Context, path paths.AbsPath) (Result, error) {
	output, err := fetchExiftoolOutput(ctx, path)
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
	if s == "" {
		return time.Time{}, errors.New("DateTimeOriginal is empty")
	}

	capturedAt, err := time.Parse(dateTimeOriginalLayout, s)
	if err != nil {
		return time.Time{}, err
	}
	return capturedAt, nil
}

func fetchExiftoolOutput(ctx context.Context, path paths.AbsPath) ([]exiftoolOutput, error) {
	cmd := exec.CommandContext(ctx, "exiftool", "-json", path.String())

	rawOutput, commandErr := cmd.Output()
	if commandErr != nil && ctx.Err() != nil {
		return nil, fmt.Errorf("running exiftool on %q: %w", path, ctx.Err())
	}

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

func isEmptyDir(path paths.AbsPath) (bool, error) {
	info, err := os.Stat(path.String())
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}

	entries, err := os.ReadDir(path.String())
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func processOutput(output []exiftoolOutput) Result {
	metadata := make([]File, 0, len(output))
	var problems []Problem

	for _, out := range output {
		out = normalizeOutput(out)

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

		fileName, err := paths.NewRelPath(out.FileName)
		if err != nil {
			problems = append(problems, Problem{
				FileName: out.FileName,
				Message:  fmt.Sprintf("invalid FileName %q: %v", out.FileName, err),
			})
			continue
		}

		capturedAt := ""
		if out.DateTimeOriginal != "" {
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

		metadata = append(metadata, File{
			FileName:    fileName,
			Format:      out.FileType,
			Title:       out.Title,
			Description: out.Description,
			CapturedAt:  capturedAt,
			Width:       out.Width,
			Height:      out.Height,
		})
	}

	slices.SortStableFunc(metadata, func(a, b File) int {
		return strings.Compare(a.FileName.String(), b.FileName.String())
	})
	slices.SortStableFunc(problems, func(a, b Problem) int {
		return strings.Compare(a.FileName, b.FileName)
	})

	return Result{
		Metadata:     metadata,
		FileProblems: problems,
	}
}

func normalizeOutput(out exiftoolOutput) exiftoolOutput {
	out.FileType = strings.TrimSpace(out.FileType)
	out.Title = strings.TrimSpace(out.Title)
	out.Description = strings.TrimSpace(out.Description)
	out.DateTimeOriginal = strings.TrimSpace(out.DateTimeOriginal)
	out.Error = strings.TrimSpace(out.Error)
	return out
}

func missingMetadata(out exiftoolOutput) []string {
	var missing []string

	if out.FileName == "" {
		missing = append(missing, "FileName")
	}
	if out.FileType == "" {
		missing = append(missing, "FileType")
	}

	return missing
}
