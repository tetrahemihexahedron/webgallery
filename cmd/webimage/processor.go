package main

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
	"tetrahemihexahedron/webimage/internal/vips"
	"time"
)

type metadataReader interface {
	Read(path string) (exif.Metadata, error)
}

type Result struct {
	DirProcessed string
	Images       []image.Processed
	Problems     []FileProblem
}

type FileProblem struct {
	FileName string
	Message  string
}

func ProcessDir(config config.Config) (Result, error) {
	inDir := config.InDir
	metadataReader := exif.Exiftool{}
	metadataResult, err := metadataReader.Read(inDir)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		DirProcessed: config.InDir,
	}

	for _, problem := range metadataResult.FileProblems {
		result.Problems = append(result.Problems, FileProblem{
			FileName: problem.FileName,
			Message:  problem.Message,
		})
	}

	for _, metadata := range metadataResult.Metadata {
		if image.ParseFormat(metadata.Format) != image.FormatJPEG {
			result.Problems = append(result.Problems, FileProblem{
				FileName: metadata.FileName,
				Message: fmt.Sprintf(
					"skipping file %q: format is %s, not JPEG",
					metadata.FileName,
					metadata.Format,
				),
			})
			continue
		}

		imageProcessed, err := processFile(metadata, config)

		if err != nil {
			result.Problems = append(result.Problems, FileProblem{
				FileName: metadata.FileName,
				Message:  fmt.Sprintf("file processing error: %v", err),
			})
		} else {
			result.Images = append(result.Images, imageProcessed)
		}
	}
	return result, nil
}

func processFile(metadata exif.Metadata, config config.Config) (image.Processed, error) {
	source := filepath.Join(config.InDir, metadata.FileName)
	imageDir := filepath.Join(config.OutDir, imageDir())

	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return image.Processed{}, fmt.Errorf("unable to make image directory %s: %w", imageDir, err)
	}

	hash, err := hashFile(source)
	if err != nil {
		deleteRemnants(imageDir)
		return image.Processed{}, fmt.Errorf("unable to hash file %s: %w", source, err)
	}

	sourceDest := filepath.Join(imageDir, "orig.jpg")
	if err = copyFile(source, sourceDest); err != nil {
		deleteRemnants(imageDir)
		return image.Processed{}, fmt.Errorf("unable to copy source %s to %s: %w", source, imageDir, err)
	}

	sourceFile := image.Source{
		Hash:   hash,
		Path:   sourceDest,
		Width:  metadata.Width,
		Height: metadata.Height,
	}

	processedImg := image.Processed{
		Source: sourceFile,
		Dir:    imageDir,
	}

	specs := variantSpecs(processedImg)
	result, err := vips.Generate(source, specs)

	if len(result.Generated()) == 0 {
		deleteRemnants(imageDir)
		if err == nil {
			err = fmt.Errorf("%d variants were attempted, and no errors were reported", len(specs))
		}
		return image.Processed{}, fmt.Errorf("no variants were generated: %w", err)
	}

	processedImg.Variants = identifyVariants(result.Generated())

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

func variantSpecs(img image.Processed) []vips.Spec {
	var desiredWidths = []int{400, 800, 1200, 1600}
	var desiredExts = []string{".jpg", ".avif"}

	// widths generated are <= the source's width
	widths := variantWidths(img.Source.Width, desiredWidths)
	specs := make([]vips.Spec, 0, len(widths)*len(desiredExts))

	for _, ext := range desiredExts {
		for _, width := range widths {
			filename := filename(width, ext)
			filepath := filepath.Join(img.Dir, filename)
			specs = append(specs, vips.Spec{OutPath: filepath, Width: width})
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

func identifyVariants(specs []vips.Spec) []image.Variant {
	variants := make([]image.Variant, 0, len(specs))
	for _, spec := range specs {
		variants = append(variants, image.Variant{
			Path:   spec.OutPath,
			Format: image.ParseFormat(filepath.Ext(spec.OutPath)),
			Width:  spec.Width,
		})
	}
	return variants
}
