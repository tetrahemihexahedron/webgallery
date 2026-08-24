package image

import (
	"testing"
	"time"
)

func TestParseCapturedAt(t *testing.T) {
	got, err := ParseCapturedAt("2024-05-12T14:22:00")
	if err != nil {
		t.Fatalf("ParseCapturedAt returned an unexpected error: %v", err)
	}

	want := time.Date(2024, time.May, 12, 14, 22, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseCapturedAt returned %v, want %v", got, want)
	}
}

func TestParseCapturedAtRejectsEmptyValue(t *testing.T) {
	_, err := ParseCapturedAt("  \t")
	if err == nil {
		t.Fatal("ParseCapturedAt returned nil error for empty value")
	}
}

func TestParseCapturedAtRejectsInvalidValue(t *testing.T) {
	_, err := ParseCapturedAt("2024:05:12 14:22:00")
	if err == nil {
		t.Fatal("ParseCapturedAt returned nil error for invalid value")
	}
}

func TestFormatCapturedAt(t *testing.T) {
	capturedAt := time.Date(2024, time.May, 12, 14, 22, 0, 0, time.FixedZone("Mountain", -6*60*60))

	got := FormatCapturedAt(capturedAt)
	want := "2024-05-12T14:22:00"
	if got != want {
		t.Errorf("FormatCapturedAt returned %q, want %q", got, want)
	}
}
