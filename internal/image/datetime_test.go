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
			if formatted := image.FormatCapturedAt(got); formatted != tc.value {
				t.Errorf("image.FormatCapturedAt(image.ParseCapturedAt(%q)) = %q, want %q", tc.value, formatted, tc.value)
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
			value: "",
		},
		{
			name:  "surrounding whitespace",
			value: "  2024-05-12T14:22:00\t",
		},
		{
			name:  "whitespace-only value",
			value: "  \t",
		},
		{
			name:  "invalid value",
			value: "2024:05:12 14:22:00",
		},
		{
			name:  "fractional seconds",
			value: "2024-05-12T14:22:00.123",
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

func TestParseProcessedAt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{
			name:  "valid UTC value",
			value: "2026-08-24T18:00:00Z",
			want:  time.Date(2026, time.August, 24, 18, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := image.ParseProcessedAt(tc.value)
			if err != nil {
				t.Fatalf("image.ParseProcessedAt(%q) returned error: %v", tc.value, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("image.ParseProcessedAt(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseProcessedAtReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "empty value",
			value: "",
		},
		{
			name:  "surrounding whitespace",
			value: "  2026-08-24T18:00:00Z\t",
		},
		{
			name:  "invalid value",
			value: "2026-08-24T18:00:00",
		},
		{
			name:  "non-UTC offset",
			value: "2026-08-24T12:00:00-06:00",
		},
		{
			name:  "fractional seconds",
			value: "2026-08-24T18:00:00.123Z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := image.ParseProcessedAt(tc.value)
			if err == nil {
				t.Fatalf("image.ParseProcessedAt(%q) returned nil error, want error", tc.value)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		format func(time.Time) string
		value  time.Time
		want   string
	}{
		{
			name:   "FormatCapturedAt preserves wall-clock time",
			format: image.FormatCapturedAt,
			value:  time.Date(2024, time.May, 12, 14, 22, 0, 0, time.FixedZone("Mountain", -6*60*60)),
			want:   "2024-05-12T14:22:00",
		},
		{
			name:   "FormatProcessedAt converts to UTC RFC3339",
			format: image.FormatProcessedAt,
			value:  time.Date(2024, time.May, 12, 14, 22, 0, 123, time.FixedZone("Mountain", -6*60*60)),
			want:   "2024-05-12T20:22:00Z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.format(tc.value)
			if got != tc.want {
				t.Errorf("format(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
