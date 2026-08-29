package paths_test

import (
	"path/filepath"
	"testing"

	"tetrahemihexahedron/webimage/internal/paths"
)

// Note: These tests assume Unix-style paths and are not Windows compatible.

func TestNewAbsPath(t *testing.T) {
	absPath := t.TempDir()
	basePath := t.TempDir()
	dirtyAbsPath := basePath + "/../" + filepath.Base(basePath)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute path succeeds",
			path: absPath,
			want: absPath,
		},
		{
			name: "cleans absolute path",
			path: dirtyAbsPath,
			want: basePath,
		},
		{
			name:    "relative path fails",
			path:    "photos/dog.jpg",
			wantErr: true,
		},
		{
			name:    "empty path fails",
			path:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := paths.NewAbsPath(tc.path)
			if err != nil {
				if tc.wantErr {
					return
				}
				t.Fatalf("paths.NewAbsPath(%q) error = %v, want nil", tc.path, err)
			}

			if tc.wantErr {
				t.Fatalf("paths.NewAbsPath(%q) error = nil, want error", tc.path)
			}

			if got.String() != tc.want {
				t.Fatalf("paths.NewAbsPath(%q).String() = %q, want %q", tc.path, got.String(), tc.want)
			}
		})
	}
}

func TestNewRelPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "relative path succeeds",
			path: "photos/dog.jpg",
			want: "photos/dog.jpg",
		},
		{
			name: "cleans relative path",
			path: "photos/../dog.jpg",
			want: "dog.jpg",
		},
		{
			name: ". path succeeds",
			path: ".",
			want: ".",
		},
		{
			name:    "absolute path fails",
			path:    t.TempDir(),
			wantErr: true,
		},
		{
			name:    "path starting with '..' fails",
			path:    "../dog.jpg",
			wantErr: true,
		},
		{
			name:    "empty path fails",
			path:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := paths.NewRelPath(tc.path)
			if err != nil {
				if tc.wantErr {
					return
				}
				t.Fatalf("paths.NewRelPath(%q) error = %v, want nil", tc.path, err)
			}

			if tc.wantErr {
				t.Fatalf("paths.NewRelPath(%q) error = nil, want error", tc.path)
			}

			if got.String() != tc.want {
				t.Fatalf("paths.NewRelPath(%q).String() = %q, want %q", tc.path, got.String(), tc.want)
			}
		})
	}
}
