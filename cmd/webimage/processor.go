package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"tetrahemihexahedron/webimage/internal/config"
	"tetrahemihexahedron/webimage/internal/image"
	"tetrahemihexahedron/webimage/internal/index"
	"tetrahemihexahedron/webimage/internal/manifest"
	"tetrahemihexahedron/webimage/internal/metadata"
	"tetrahemihexahedron/webimage/internal/paths"
	"tetrahemihexahedron/webimage/internal/variants"
)

type metadataReader interface {
	Read(path paths.AbsPath) (metadata.Result, error)
}

type variantGenerator interface {
	Generate(source paths.AbsPath, specs []variants.Spec) (variants.Result, error)
}

type result struct {
	dirProcessed paths.AbsPath
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
	inDirAbsPath := p.cfg.InDir.String()

	fmt.Fprintf(p.progressReporter, "Processing image files in %q\n", inDirAbsPath)

	imageIndex, err := index.Read(p.cfg.OutDir)
	if err != nil {
		return result{}, err
	}

	imageDirsByHash, err := imageIndex.ImageDirsBySHA256()
	if err != nil {
		return result{}, err
	}

	metadataResult, err := p.metadataReader.Read(p.cfg.InDir)
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
		dirProcessed: p.cfg.InDir,
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

		sourceAbsPath, err := paths.NewAbsPath(filepath.Join(p.cfg.InDir.String(), metadata.FileName))
		if err != nil {
			fmt.Fprintf(
				p.progressReporter,
				"Error building path for %q: %v\n",
				metadata.FileName,
				err,
			)

			result.problems = append(result.problems, fileProblem{
				fileName: metadata.FileName,
				message:  fmt.Sprintf("source path error: %v", err),
			})
			continue
		}

		sourceHash, err := hashFile(sourceAbsPath)
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

		if existingImgDir, ok := imageDirsByHash[sourceHash]; ok {
			fmt.Fprintf(
				p.progressReporter,
				"Skipping %q: duplicate of image in %q\n",
				metadata.FileName,
				existingImgDir,
			)

			result.problems = append(result.problems, fileProblem{
				fileName: metadata.FileName,
				message:  fmt.Sprintf("skipping duplicate of image in %q", existingImgDir),
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
			imageDirsByHash[sourceHash] = imageProcessed.DirRelPath

			fmt.Fprintf(
				p.progressReporter,
				"Processed %q:\n\t%d variants generated in %q\n",
				metadata.FileName,
				len(imageProcessed.Variants),
				imageProcessed.DirAbsPath,
			)

			result.images = append(result.images, imageProcessed)
		}
	}
	return result, nil
}

func (p *processor) processFile(metadata image.Metadata, sourceHash string) (image.Processed, error) {
	processedAt := time.Now().UTC()
	sourceAbsPath, err := paths.NewAbsPath(filepath.Join(p.cfg.InDir.String(), metadata.FileName))
	if err != nil {
		return image.Processed{}, err
	}
	dirDate, err := dirDate(p.cfg.DirDate, metadata.CapturedAt, processedAt)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirRelPath, err := imgDirRelPath(dirDate)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirAbsPath, err := paths.NewAbsPath(filepath.Join(p.cfg.OutDir.String(), imgDirRelPath.String()))
	if err != nil {
		return image.Processed{}, err
	}

	if err := os.MkdirAll(imgDirAbsPath.String(), 0755); err != nil {
		return image.Processed{}, fmt.Errorf("unable to make image directory %s: %w", imgDirAbsPath, err)
	}

	sourceDest, err := paths.NewAbsPath(filepath.Join(imgDirAbsPath.String(), "orig.jpg"))
	if err != nil {
		deleteRemnants(imgDirAbsPath)
		return image.Processed{}, err
	}
	if err = copyFile(sourceAbsPath, sourceDest); err != nil {
		deleteRemnants(imgDirAbsPath)
		return image.Processed{}, fmt.Errorf("unable to copy source %s to %s: %w", sourceAbsPath, imgDirAbsPath, err)
	}

	sourceFile := image.Source{
		Hash:   sourceHash,
		Path:   sourceDest.String(),
		Width:  metadata.Width,
		Height: metadata.Height,
	}

	processedImg := image.Processed{
		Source:      sourceFile,
		DirAbsPath:  imgDirAbsPath,
		DirRelPath:  imgDirRelPath,
		Title:       metadata.Title,
		Description: metadata.Description,
		CapturedAt:  metadata.CapturedAt,
		ProcessedAt: image.FormatProcessedAt(processedAt),
	}

	specs, err := variantSpecs(processedImg)
	if err != nil {
		deleteRemnants(imgDirAbsPath)
		return image.Processed{}, err
	}
	result, err := p.variantGenerator.Generate(sourceAbsPath, specs)

	if len(result.Generated) == 0 {
		deleteRemnants(imgDirAbsPath)
		if err == nil {
			err = fmt.Errorf("%d variants were attempted, and no errors were reported", len(specs))
		}
		return image.Processed{}, fmt.Errorf("no variants were generated: %w", err)
	}

	processedImg.Variants, err = identifyVariants(result.Generated)
	if err != nil {
		deleteRemnants(imgDirAbsPath)
		return image.Processed{}, fmt.Errorf("unable to identify generated variants: %w", err)
	}

	if err := manifest.Write(processedImg); err != nil {
		deleteRemnants(imgDirAbsPath)
		return image.Processed{}, fmt.Errorf("unable to write manifest: %w", err)
	}

	return processedImg, nil
}

func hashFile(path paths.AbsPath) (string, error) {
	f, err := os.Open(path.String())
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

func imgDirRelPath(date time.Time) (paths.RelPath, error) {
	randId := ""
	b := make([]byte, 7)
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	for !re.MatchString(randId) {
		if _, err := rand.Read(b); err != nil {
			return paths.RelPath{}, fmt.Errorf("generating random image directory ID: %w", err)
		}
		randId = base64.RawURLEncoding.EncodeToString(b)
	}

	path, err := paths.NewRelPath(fmt.Sprintf("%d/%02d/%s", date.Year(), date.Month(), randId))
	if err != nil {
		return paths.RelPath{}, fmt.Errorf("building image directory relative path: %w", err)
	}

	return path, nil
}

func variantSpecs(img image.Processed) ([]variants.Spec, error) {
	var desiredWidths = []int{400, 800, 1200, 1600}
	var desiredExts = []string{".jpg", ".avif"}

	// widths generated are <= the source's width
	widths := variantWidths(img.Source.Width, desiredWidths)
	specs := make([]variants.Spec, 0, len(widths)*len(desiredExts))

	for _, ext := range desiredExts {
		for _, width := range widths {
			filename := filename(width, ext)
			outPath, err := paths.NewAbsPath(filepath.Join(img.DirAbsPath.String(), filename))
			if err != nil {
				return nil, err
			}
			specs = append(specs, variants.Spec{OutPath: outPath, Width: width})
		}
	}
	return specs, nil
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

func copyFile(source paths.AbsPath, dest paths.AbsPath) error {
	sourcefile, err := os.Open(source.String())
	if err != nil {
		return err
	}
	defer sourcefile.Close()

	destination, err := os.Create(dest.String())
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, sourcefile)
	return err
}

func deleteRemnants(dir paths.AbsPath) error {
	return os.RemoveAll(dir.String())
}

func identifyVariants(specs []variants.Spec) ([]image.Variant, error) {
	variants := make([]image.Variant, 0, len(specs))
	for _, spec := range specs {
		filename := filepath.Base(spec.OutPath.String())
		path, err := paths.NewRelPath(filename)
		if err != nil {
			return nil, fmt.Errorf("determining relative path for %s: %w", spec.OutPath, err)
		}

		variants = append(variants, image.Variant{
			Path:   path,
			Format: image.ParseFormat(filepath.Ext(filename)),
			Width:  spec.Width,
		})
	}
	return variants, nil
}
