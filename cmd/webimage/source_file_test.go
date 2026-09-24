package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	sourceContents := []byte("source image contents")

	t.Run("new destination", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := filepath.Join(dir, "source.jpg")
		destPath := filepath.Join(dir, "destination.jpg")
		if err := os.WriteFile(sourcePath, sourceContents, 0600); err != nil {
			t.Fatalf("os.WriteFile(%q) returned error: %v", sourcePath, err)
		}

		if err := copyFile(mustAbs(t, sourcePath), mustAbs(t, destPath)); err != nil {
			t.Fatalf("copyFile(%q, %q) returned error: %v", sourcePath, destPath, err)
		}
		got, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) returned error: %v", destPath, err)
		}
		if !bytes.Equal(got, sourceContents) {
			t.Errorf("copyFile(%q, %q) contents = %q, want %q", sourcePath, destPath, got, sourceContents)
		}
	})

	existingContents := []byte("existing destination contents")
	tests := []struct {
		name        string
		prepareDest func(sourcePath, destPath string) (string, error)
		wantDest    []byte
	}{
		{
			name: "same path",
			prepareDest: func(sourcePath, _ string) (string, error) {
				return sourcePath, nil
			},
			wantDest: sourceContents,
		},
		{
			name: "hard-link alias",
			prepareDest: func(sourcePath, destPath string) (string, error) {
				return destPath, os.Link(sourcePath, destPath)
			},
			wantDest: sourceContents,
		},
		{
			name: "symlink alias",
			prepareDest: func(sourcePath, destPath string) (string, error) {
				return destPath, os.Symlink(sourcePath, destPath)
			},
			wantDest: sourceContents,
		},
		{
			name: "unrelated pre-existing destination",
			prepareDest: func(_, destPath string) (string, error) {
				return destPath, os.WriteFile(destPath, existingContents, 0644)
			},
			wantDest: existingContents,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sourcePath := filepath.Join(dir, "source.jpg")
			if err := os.WriteFile(sourcePath, sourceContents, 0600); err != nil {
				t.Fatalf("os.WriteFile(%q) returned error: %v", sourcePath, err)
			}
			destPath, err := tc.prepareDest(sourcePath, filepath.Join(dir, "destination.jpg"))
			if err != nil {
				t.Fatalf("preparing destination: %v", err)
			}

			err = copyFile(mustAbs(t, sourcePath), mustAbs(t, destPath))
			if err == nil {
				t.Fatalf("copyFile(%q, %q) returned nil error, want error", sourcePath, destPath)
			}

			gotSource, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) returned error: %v", sourcePath, err)
			}
			if !bytes.Equal(gotSource, sourceContents) {
				t.Errorf("source contents after copyFile() = %q, want %q", gotSource, sourceContents)
			}
			gotDest, err := os.ReadFile(destPath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) returned error: %v", destPath, err)
			}
			if !bytes.Equal(gotDest, tc.wantDest) {
				t.Errorf("destination contents after copyFile() = %q, want %q", gotDest, tc.wantDest)
			}
		})
	}
}
