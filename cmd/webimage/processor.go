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
	"time"

	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/variants"
)

type metadataReader interface {
	Read(path string) (metadata.Result, error)
}

type variantGenerator interface {
	Generate(source string, specs []variants.Spec) (variants.Result, error)
}

type result struct {
	dirProcessed string
	images       []image.Processed
	problems     []fileProblem
}

type fileProblem struct {
	fileName string
	message  string
}

type processor struct {
	cfg              config.Config
	metadataReader   metadataReader
	variantGenerator variantGenerator
	progressReporter io.Writer
}

func (p *processor) processDir() (result, error) {
	inDir := p.cfg.InDir

	fmt.Fprintf(p.progressReporter, "Processing image files in %q\n", inDir)

	metadataResult, err := p.metadataReader.Read(inDir)
	if err != nil {
		return result{}, err
	}

	fmt.Fprintf(
		p.progressReporter,
		"Read metadata from %d file(s) with %d error(s)\n",
		len(metadataResult.Metadata)+len(metadataResult.FileProblems),
		len(metadataResult.FileProblems),
	)

	result := result{
		dirProcessed: inDir,
	}

	for _, problem := range metadataResult.FileProblems {
		fmt.Fprintf(
			p.progressReporter,
			"\t%q: %s\n",
			problem.FileName,
			problem.Message,
		)

		result.problems = append(result.problems, fileProblem{
			fileName: problem.FileName,
			message:  problem.Message,
		})
	}

	fmt.Fprint(p.progressReporter, "\n----------------\n")

	for _, metadata := range metadataResult.Metadata {
		if image.ParseFormat(metadata.Format) != image.FormatJPEG {
			fmt.Fprintf(
				p.progressReporter,
				"Skipping %q: format is %s, not JPEG\n",
				metadata.FileName,
				metadata.Format,
			)

			result.problems = append(result.problems, fileProblem{
				fileName: metadata.FileName,
				message: fmt.Sprintf(
					"skipping file %q: format is %s, not JPEG",
					metadata.FileName,
					metadata.Format,
				),
			})
			continue
		}

		source := filepath.Join(p.cfg.InDir, metadata.FileName)
		sourceHash, err := hashFile(source)
		if err != nil {
			fmt.Fprintf(
				p.progressReporter,
				"Error hashing %q: %v\n",
				metadata.FileName,
				err,
			)

			result.problems = append(result.problems, fileProblem{
				fileName: metadata.FileName,
				message:  fmt.Sprintf("file hashing error: %v", err),
			})
			continue
		}

		imageProcessed, err := p.processFile(metadata, sourceHash)

		if err != nil {
			fmt.Fprintf(
				p.progressReporter,
				"Error processing %q: %v\n",
				metadata.FileName,
				err,
			)

			result.problems = append(result.problems, fileProblem{
				fileName: metadata.FileName,
				message:  fmt.Sprintf("file processing error: %v", err),
			})
		} else {
			fmt.Fprintf(
				p.progressReporter,
				"Processed %q:\n\t%d variants generated in %q\n",
				metadata.FileName,
				len(imageProcessed.Variants),
				imageProcessed.Dir,
			)

			result.images = append(result.images, imageProcessed)
		}
	}
	return result, nil
}

func (p *processor) processFile(metadata image.Metadata, sourceHash string) (image.Processed, error) {
	processedAt := time.Now().UTC()
	source := filepath.Join(p.cfg.InDir, metadata.FileName)
	dirDate, err := dirDate(p.cfg.DirDate, metadata.CapturedAt, processedAt)
	if err != nil {
		return image.Processed{}, err
	}
	imageDir := filepath.Join(p.cfg.OutDir, imageDir(dirDate))

	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return image.Processed{}, fmt.Errorf("unable to make image directory %s: %w", imageDir, err)
	}

	sourceDest := filepath.Join(imageDir, "orig.jpg")
	if err = copyFile(source, sourceDest); err != nil {
		deleteRemnants(imageDir)
		return image.Processed{}, fmt.Errorf("unable to copy source %s to %s: %w", source, imageDir, err)
	}

	sourceFile := image.Source{
		Hash:   sourceHash,
		Path:   sourceDest,
		Width:  metadata.Width,
		Height: metadata.Height,
	}

	processedImg := image.Processed{
		Source:      sourceFile,
		Dir:         imageDir,
		Title:       metadata.Title,
		Description: metadata.Description,
		CapturedAt:  metadata.CapturedAt,
		ProcessedAt: image.FormatProcessedAt(processedAt),
	}

	specs := variantSpecs(processedImg)
	result, err := p.variantGenerator.Generate(source, specs)

	if len(result.Generated) == 0 {
		deleteRemnants(imageDir)
		if err == nil {
			err = fmt.Errorf("%d variants were attempted, and no errors were reported", len(specs))
		}
		return image.Processed{}, fmt.Errorf("no variants were generated: %w", err)
	}

	processedImg.Variants = identifyVariants(result.Generated)

	if err := manifest.Write(processedImg); err != nil {
		deleteRemnants(imageDir)
		return image.Processed{}, fmt.Errorf("unable to write manifest: %w", err)
	}

	return processedImg, nil
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

func dirDate(dirDate config.DirDate, capturedAt string, processedAt time.Time) (time.Time, error) {
	switch dirDate {
	case config.DirDateProcessed:
		return processedAt, nil
	case config.DirDateCaptured:
		capturedDate, err := image.ParseCapturedAt(capturedAt)
		if err != nil {
			return time.Time{}, fmt.Errorf("--dir-date=captured requires a valid capturedAt: %w", err)
		}
		return capturedDate, nil
	default:
		return time.Time{}, fmt.Errorf("invalid directory date %q", dirDate)
	}
}

func imageDir(date time.Time) string {
	randId := ""
	b := make([]byte, 7)
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	for !re.MatchString(randId) {
		if _, err := rand.Read(b); err != nil {
			log.Fatal(err)
		}
		randId = base64.RawURLEncoding.EncodeToString(b)
	}

	return fmt.Sprintf("%d/%02d/%s/", date.Year(), date.Month(), randId)
}

func variantSpecs(img image.Processed) []variants.Spec {
	var desiredWidths = []int{400, 800, 1200, 1600}
	var desiredExts = []string{".jpg", ".avif"}

	// widths generated are <= the source's width
	widths := variantWidths(img.Source.Width, desiredWidths)
	specs := make([]variants.Spec, 0, len(widths)*len(desiredExts))

	for _, ext := range desiredExts {
		for _, width := range widths {
			filename := filename(width, ext)
			outPath := filepath.Join(img.Dir, filename)
			specs = append(specs, variants.Spec{OutPath: outPath, Width: width})
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

func identifyVariants(specs []variants.Spec) []image.Variant {
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
