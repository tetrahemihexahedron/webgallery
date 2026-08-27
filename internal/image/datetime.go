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
	return capturedAt, nil
}

func FormatCapturedAt(t time.Time) string {
	return t.Format(capturedAtLayout)
}

func FormatProcessedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
