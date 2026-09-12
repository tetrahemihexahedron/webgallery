package image

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const capturedAtLayout = "2006-01-02T15:04:05"

func ParseCapturedAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("capturedAt is empty")
	}

	capturedAt, err := time.Parse(capturedAtLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing capturedAt %q: %w", s, err)
	}
	if FormatCapturedAt(capturedAt) != s {
		return time.Time{}, fmt.Errorf("capturedAt %q must be in webimage's normalized capturedAt format", s)
	}
	return capturedAt, nil
}

func FormatCapturedAt(t time.Time) string {
	return t.Format(capturedAtLayout)
}

func ParseProcessedAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("processedAt is empty")
	}

	processedAt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing processedAt %q: %w", s, err)
	}
	if FormatProcessedAt(processedAt) != s {
		return time.Time{}, fmt.Errorf("processedAt %q must be UTC RFC3339", s)
	}
	return processedAt, nil
}

func FormatProcessedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
