package gallery

import (
	"errors"
	"fmt"
	"io"

	"tetrahemihexahedron/webimage/internal/paths"
)

// SortField selects which image datetime field controls display order.
type SortField string

const (
	SortCaptured  SortField = "captured"
	SortProcessed SortField = "processed"
)

// IsValid reports whether s is a supported gallery sort field.
func (s SortField) IsValid() bool {
	switch s {
	case SortCaptured, SortProcessed:
		return true
	default:
		return false
	}
}

// Options configures gallery rendering.
type Options struct {
	ImagesRoot paths.AbsPath
	URLPrefix  string
	Sort       SortField
}

// Render writes gallery HTML to w.
func Render(w io.Writer, opts Options) error {
	if w == nil {
		return errors.New("writer can't be nil")
	}
	if !opts.Sort.IsValid() {
		return fmt.Errorf("invalid sort field %q", opts.Sort)
	}

	return errors.New("gallery rendering is not implemented")
}
