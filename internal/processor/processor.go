package processor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/exif"
	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/variant"
	"time"
)

func ProcessDir(config config.Config) error {
	inDir := config.InDir
	allMetadata, errs := exif.FetchMetadata(inDir)

	log.Printf("Fetched metadata for %d files. %d error(s).", len(allMetadata), len(errs))
	for _, err := range errs {
		log.Printf("Metadata error: %v", err)
	}

	for _, metadata := range allMetadata {
		if metadata.FileType != image.FormatJPEG {
			log.Printf("Skipping file %s: file type is %s, not JPEG", metadata.FileName, metadata.FileType)
			continue
		}

		_, err := processFile(metadata, config)

		if err != nil {
			log.Printf("File processing error: %v", err)
		}
	}
	return nil
}

func processFile(metadata image.Metadata, config config.Config) (image.Processed, error) {
	source := filepath.Join(config.InDir, metadata.FileName)
	imageDir := filepath.Join(config.OutDir, imageDir())

	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return image.Processed{}, fmt.Errorf("unable to make image directory %s: %w", imageDir, err)
	}

	hash, err := hashFile(source)
	if err != nil {
		return image.Processed{}, fmt.Errorf("unable to hash file %s: %w", source, err)
	}

	if err = copyFile(source, filepath.Join(imageDir, "orig.jpg")); err != nil {
		deleteRemnants(imageDir)
		return image.Processed{}, fmt.Errorf("unable to copy source %s to %s: %w", source, imageDir, err)
	}

	processedImg := image.Processed{
		ImageDir: imageDir,
		Hash:     hash,
		Metadata: metadata,
	}

	specs := variantSpecs(processedImg)
	result, err := variant.Generate(source, specs)
	log.Printf(
		"result: %d variant(s) generated; %d error(s)",
		len(result.Generated()),
		len(result.Failed()),
	)
	return processedImg, err
}

func hashFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func imageDir() string {
	randId := ""
	b := make([]byte, 7)
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	for !re.MatchString(randId) {
		if _, err := rand.Read(b); err != nil {
			log.Fatal(err)
		}
		randId = base64.RawURLEncoding.EncodeToString(b)
	}

	year := time.Now().Year()
	month := time.Now().Month()

	return fmt.Sprintf("%d/%02d/%s/", year, month, randId)
}

func variantSpecs(img image.Processed) []variant.Spec {
	var desiredWidths = []int{400, 800, 1200, 1600}
	var desiredExts = []string{".jpg", ".avif"}

	// widths generated are <= the source's width
	widths := variantWidths(img.Metadata.Width, desiredWidths)
	specs := make([]variant.Spec, 0, len(widths)*len(desiredExts))

	for _, ext := range desiredExts {
		for _, width := range widths {
			filename := filename(width, ext)
			filepath := filepath.Join(img.ImageDir, filename)
			specs = append(specs, variant.Spec{OutPath: filepath, Width: width})
		}
	}
	return specs
}

func variantWidths(sourceWidth int, desired []int) []int {
	widths := make([]int, 0, len(desired))

	for _, width := range desired {
		if sourceWidth <= width {
			widths = append(widths, sourceWidth)
			return widths
		}
		widths = append(widths, width)
	}
	return widths
}

func filename(width int, ext string) string {
	return "w" + strconv.Itoa(width) + ext
}

func copyFile(source string, dest string) error {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, sourcefile)
	return err
}

func deleteRemnants(dir string) error {
	return os.RemoveAll(dir)
}
