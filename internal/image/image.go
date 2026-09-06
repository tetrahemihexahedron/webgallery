package image

import "tetrahemihexahedron/webimage/internal/paths"

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
	Path   paths.RelPath
	Format Format
	Width  int
}

type Processed struct {
	Source      Source
	DirAbsPath  paths.AbsPath
	DirRelPath  paths.RelPath
	Title       string
	Description string
	CapturedAt  string
	ProcessedAt string
	Variants    []Variant
}
