package image_test

import (
	"testing"

	"tetrahemihexahedron/webimage/internal/image"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  image.Format
	}{
		{
			name:  "JPEG",
			value: "JPEG",
			want:  image.FormatJPEG,
		},
		{
			name:  "JPG extension",
			value: ".jpg",
			want:  image.FormatJPEG,
		},
		{
			name:  "AVIF",
			value: "AVIF",
			want:  image.FormatAVIF,
		},
		{
			name:  "AVIF extension",
			value: ".avif",
			want:  image.FormatAVIF,
		},
		{
			name:  "unsupported format",
			value: "PNG",
			want:  image.FormatOther,
		},
		{
			name:  "empty format",
			value: "",
			want:  image.FormatOther,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := image.ParseFormat(tc.value)
			if got != tc.want {
				t.Errorf("image.ParseFormat(%q) = %s, want %s", tc.value, got, tc.want)
			}
		})
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		name   string
		format image.Format
		want   string
	}{
		{name: "JPEG", format: image.FormatJPEG, want: "JPEG"},
		{name: "AVIF", format: image.FormatAVIF, want: "AVIF"},
		{name: "OTHER", format: image.FormatOther, want: "OTHER"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.format.String()
			if got != tc.want {
				t.Errorf("image.Format(%q).String() = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
