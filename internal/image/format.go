package image

import "strings"

type Format string

const (
	FormatJPEG  Format = "JPEG"
	FormatAVIF  Format = "AVIF"
	FormatOther Format = "OTHER"
)

func (f Format) String() string {
	return string(f)
}

func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "jpeg", "jpg", ".jpeg", ".jpg":
		return FormatJPEG
	case "avif", ".avif":
		return FormatAVIF
	default:
		return FormatOther
	}
}
