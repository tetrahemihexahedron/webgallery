package main

import (
	"io"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want config
	}{
		{
			name: "required flags only",
			args: []string{"-incoming", "incoming", "-output", "output"},
			want: config{
				inDir:   mustAbs(t, "incoming"),
				outDir:  mustAbs(t, "output"),
				dirDate: dirDateProcessed,
			},
		},
		{
			name: "quiet",
			args: []string{"-incoming", "incoming", "-output", "output", "-quiet"},
			want: config{
				inDir:   mustAbs(t, "incoming"),
				outDir:  mustAbs(t, "output"),
				isQuiet: true,
				dirDate: dirDateProcessed,
			},
		},
		{
			name: "captured dir date",
			args: []string{"-incoming", "incoming", "-output", "output", "-dir-date", "captured"},
			want: config{
				inDir:   mustAbs(t, "incoming"),
				outDir:  mustAbs(t, "output"),
				dirDate: dirDateCaptured,
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

func TestValidateRoots(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		output   string
		wantErr  bool
	}{
		{name: "equal roots", incoming: "/srv/photos", output: "/srv/photos", wantErr: true},
		{name: "output beneath input", incoming: "/srv/photos", output: "/srv/photos/output", wantErr: true},
		{name: "input beneath output", incoming: "/srv/photos/incoming", output: "/srv/photos", wantErr: true},
		{name: "siblings", incoming: "/srv/photos/incoming", output: "/srv/photos/output"},
		{name: "prefix-similar names", incoming: "/srv/photos", output: "/srv/photos-output"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRoots(mustAbs(t, tc.incoming), mustAbs(t, tc.output))
			if tc.wantErr && err == nil {
				t.Fatal("validateRoots() error = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateRoots() error = %v, want nil", err)
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
		{
			name:    "overlapping roots",
			args:    []string{"-incoming", "photos", "-output", "photos/output"},
			wantErr: "overlap",
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
