package index

// Index describes the processed images in a directory.
type Index struct {
	GeneratedAt string  `json:"generatedAt"`
	Images      []Image `json:"images"`
}

// Image is one processed image entry in an Index.
type Image struct {
	Dir         string `json:"dir"`
	Manifest    string `json:"manifest"`
	Title       string `json:"title"`
	CapturedAt  string `json:"capturedAt"`
	ProcessedAt string `json:"processedAt"`
	SHA256      string `json:"sha256"`
}
