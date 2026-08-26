package image

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
