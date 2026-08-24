package image

import (
	"strings"
)

type Metadata struct {
	FileName    string
	Format      string
	Title       string
	Description string
	CapturedAt  string
	Width       int
	Height      int
}

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
	Source      Source
	Dir         string
	Title       string
	Description string
	CapturedAt  string
	Variants    []Variant
}

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
