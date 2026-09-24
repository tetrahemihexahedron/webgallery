package main

import (
	"path/filepath"
	"testing"

	"tetrahemihexahedron/webgallery/internal/paths"
)

func mustAbs(t *testing.T, path string) paths.AbsPath {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v, want nil", path, err)
	}

	p, err := paths.NewAbsPath(absPath)
	if err != nil {
		t.Fatalf("paths.NewAbsPath(%q) error = %v, want nil", absPath, err)
	}

	return p
}

func mustRel(t *testing.T, path string) paths.RelPath {
	t.Helper()

	p, err := paths.NewRelPath(path)
	if err != nil {
		t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", path, err)
	}

	return p
}
