package image

import (
	"strings"
)

type Source struct {
	Hash   string
	Path   string
	Width  int
	Height int
}

type Variant struct {
	Path   string
	Format Format
	Width  int
}

type Processed struct {
	Source   Source
	Dir      string
	Variants []Variant
}

type Format string

const (
	FormatJPEG  Format = "JPEG"
	FormatAVIF  Format = "AVIF"
	FormatOther Format = "OTHER"
)

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
