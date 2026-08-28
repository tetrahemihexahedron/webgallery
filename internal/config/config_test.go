package config

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "required flags only",
			args: []string{"-incoming", "incoming", "-output", "output"},
			want: Config{
				InDir:   mustAbs(t, "incoming"),
				OutDir:  mustAbs(t, "output"),
				DirDate: DirDateProcessed,
			},
		},
		{
			name: "quiet",
			args: []string{"-incoming", "incoming", "-output", "output", "-quiet"},
			want: Config{
				InDir:   mustAbs(t, "incoming"),
				OutDir:  mustAbs(t, "output"),
				IsQuiet: true,
				DirDate: DirDateProcessed,
			},
		},
		{
			name: "captured dir date",
			args: []string{"-incoming", "incoming", "-output", "output", "-dir-date", "captured"},
			want: Config{
				InDir:   mustAbs(t, "incoming"),
				OutDir:  mustAbs(t, "output"),
				DirDate: DirDateCaptured,
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
			name:    "missing incoming",
			args:    []string{"-output", "output"},
			wantErr: "incoming",
		},
		{
			name:    "missing output",
			args:    []string{"-incoming", "incoming"},
			wantErr: "output",
		},
		{
			name:    "invalid dir date",
			args:    []string{"-incoming", "incoming", "-output", "output", "-dir-date", "modified"},
			wantErr: "dir-date",
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

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v, want nil", path, err)
	}

	return absPath
}
