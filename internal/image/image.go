package image

import "tetrahemihexahedron/webimage/internal/paths"

type Metadata struct {
	FileName    string
	Format      string
	Title       string
	Description string
	// CapturedAt is empty or exactly the value produced by FormatCapturedAt.
	CapturedAt string
	Width      int
	Height     int
}

type Source struct {
	Hash   string
	Width  int
	Height int
}

type Variant struct {
	Path   paths.RelPath
	Format Format
	Width  int
	Height int
}

type Processed struct {
	Source      Source
	DirRelPath  paths.RelPath
	Title       string
	Description string
	// CapturedAt is empty or exactly the value produced by FormatCapturedAt.
	CapturedAt  string
	ProcessedAt string
	Variants    []Variant
}
