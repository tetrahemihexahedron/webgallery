package main

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/gallery"
	"tetrahemihexahedron/webimage/internal/paths"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "required flags only",
			args: []string{"-images", "images"},
			want: Config{
				ImagesRoot: mustAbs(t, "images"),
				UseStdout:  true,
				Sort:       gallery.SortProcessed,
			},
		},
		{
			name: "all optional flags",
			args: []string{"-images", "images", "-output", "gallery.html", "-url-prefix", "/images", "-sort", "processed"},
			want: Config{
				ImagesRoot: mustAbs(t, "images"),
				OutFile:    mustAbs(t, "gallery.html"),
				UseStdout:  false,
				URLPrefix:  "/images",
				Sort:       gallery.SortProcessed,
			},
		},
		{
			name: "explicit stdout output",
			args: []string{"-images", "images", "-output", "-"},
			want: Config{
				ImagesRoot: mustAbs(t, "images"),
				UseStdout:  true,
				Sort:       gallery.SortProcessed,
			},
		},
		{
			name: "explicit captured sort",
			args: []string{"-images", "images", "-sort", "captured"},
			want: Config{
				ImagesRoot: mustAbs(t, "images"),
				UseStdout:  true,
				Sort:       gallery.SortCaptured,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args, io.Discard)
			if err != nil {
				t.Fatalf("parseArgs() error = %v, want nil", err)
			}

			if got != tc.want {
				t.Fatalf("parseArgs() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing images",
			args:    []string{},
			wantErr: "images",
		},
		{
			name:    "invalid sort",
			args:    []string{"-images", "images", "-sort", "filename"},
			wantErr: "sort",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs(tc.args, io.Discard)
			if err == nil {
				t.Fatalf("parseArgs() error = nil, want error containing %q", tc.wantErr)
			}

			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseArgs() error = %v, want message containing %q", err, tc.wantErr)
			}
		})
	}
}

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
