package variants_test

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tetrahemihexahedron/webimage/internal/paths"
)

type imageSize struct {
	width  int
	height int
}

func readImageSize(t *testing.T, path paths.AbsPath) imageSize {
	t.Helper()

	// -s3 causes exiftool to output the values of the fields only
	out, err := exec.Command("exiftool", "-ImageWidth", "-ImageHeight", "-s3", path.String()).Output()
	if err != nil {
		t.Fatalf("reading image size for %q: %v", path, err)
	}

	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		t.Fatalf("reading image size for %q returned %q, want width and height", path, out)
	}

	width, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("parsing image width for %q: %v", path, err)
	}
	height, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("parsing image height for %q: %v", path, err)
	}

	return imageSize{width: width, height: height}
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
