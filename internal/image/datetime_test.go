package image_test

import (
	"testing"
	"time"

	"tetrahemihexahedron/webimage/internal/image"
)

func TestParseCapturedAt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{
			name:  "valid value",
			value: "2024-05-12T14:22:00",
			want:  time.Date(2024, time.May, 12, 14, 22, 0, 0, time.UTC),
		},
		{
			name:  "trims whitespace",
			value: "  2024-05-12T14:22:00\t",
			want:  time.Date(2024, time.May, 12, 14, 22, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := image.ParseCapturedAt(tc.value)
			if err != nil {
				t.Fatalf("image.ParseCapturedAt(%q) returned error: %v", tc.value, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("image.ParseCapturedAt(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseCapturedAtReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "empty value",
			value: "  \t",
		},
		{
			name:  "invalid value",
			value: "2024:05:12 14:22:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := image.ParseCapturedAt(tc.value)
			if err == nil {
				t.Fatalf("image.ParseCapturedAt(%q) returned nil error, want error", tc.value)
			}
		})
	}
}

func TestFormatCapturedAt(t *testing.T) {
	capturedAt := time.Date(2024, time.May, 12, 14, 22, 0, 0, time.FixedZone("Mountain", -6*60*60))

	got := image.FormatCapturedAt(capturedAt)
	want := "2024-05-12T14:22:00"
	if got != want {
		t.Errorf("image.FormatCapturedAt(%v) = %q, want %q", capturedAt, got, want)
	}
}
