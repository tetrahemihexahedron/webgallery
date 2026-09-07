// Package paths defines small path types with validation at construction time.
package paths

import (
	"fmt"
	"path/filepath"
)

// AbsPath is a cleaned absolute filesystem path.
type AbsPath struct {
	s string
}

// NewAbsPath returns p as a cleaned AbsPath.
func NewAbsPath(p string) (AbsPath, error) {
	if !filepath.IsAbs(p) {
		return AbsPath{}, fmt.Errorf("path must be absolute: %q", p)
	}

	return AbsPath{s: filepath.Clean(p)}, nil
}

// String returns the path as a string.
func (p AbsPath) String() string {
	return p.s
}

// RelPath is a cleaned local relative filesystem path.
type RelPath struct {
	s string
}

// NewRelPath returns p as a cleaned RelPath.
//
// A RelPath must be local according to filepath.IsLocal: it cannot be empty,
// absolute, or escape its base directory with leading ".." elements.
func NewRelPath(p string) (RelPath, error) {
	if !filepath.IsLocal(p) {
		return RelPath{}, fmt.Errorf("path must be relative: %q", p)
	}

	return RelPath{s: filepath.Clean(p)}, nil
}

// String returns the path as a string.
func (p RelPath) String() string {
	return p.s
}

// JoinAbs joins base and rel and returns the result as an AbsPath.
func JoinAbs(base AbsPath, rel RelPath) (AbsPath, error) {
	if rel.String() == "" {
		return AbsPath{}, fmt.Errorf("relative path cannot be empty")
	}

	return NewAbsPath(filepath.Join(base.String(), rel.String()))
}
