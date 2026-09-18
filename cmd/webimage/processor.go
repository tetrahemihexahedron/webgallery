package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

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
	cfg              Config
	metadataReader   metadataReader
	variantGenerator variantGenerator
	progressReporter io.Writer
}

func (p *processor) processDir() (result, error) {
	inDirAbsPath := p.cfg.InDir.String()

	fmt.Fprintf(p.progressReporter, "Processing image files in %q\n", inDirAbsPath)

	imageIndex, imageDirsByHash, err := loadExistingIndex(p.cfg.OutDir)
	if err != nil {
		return result{}, err
	}

	metadataResult, err := p.readIncomingMetadata()
	if err != nil {
		return result{}, err
	}

	res := result{
		dirProcessed: p.cfg.InDir,
		problems:     p.recordMetadataProblems(metadataResult.FileProblems),
	}

	fmt.Fprint(p.progressReporter, "\n----------------\n")

	for _, metadata := range metadataResult.Metadata {
		imageProcessed, problem := p.processMetadataEntry(metadata, imageDirsByHash)
		if problem != nil {
			res.problems = append(res.problems, *problem)
			continue
		}

		res.images = append(res.images, imageProcessed)
	}

	if err := p.writeUpdatedIndex(&imageIndex, res.images); err != nil {
		return result{}, err
	}

	return res, nil
}

func (p *processor) writeUpdatedIndex(imageIndex *index.Index, images []image.Processed) error {
	if err := imageIndex.AppendAndWriteDir(p.cfg.OutDir, images); err != nil {
		updateErr := fmt.Errorf("updating index: %w", err)
		if cleanupErr := deleteProcessedImageDirs(p.cfg.OutDir, images); cleanupErr != nil {
			return errors.Join(updateErr, cleanupErr)
		}
		return updateErr
	}

	return nil
}

func (p *processor) processMetadataEntry(metadata image.Metadata, imageDirsByHash map[string]paths.RelPath) (image.Processed, *fileProblem) {
	if image.ParseFormat(metadata.Format) != image.FormatJPEG {
		fmt.Fprintf(
			p.progressReporter,
			"Skipping %q: format is %s, not JPEG\n",
			metadata.FileName,
			metadata.Format,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message: fmt.Sprintf(
				"skipping file %q: format is %s, not JPEG",
				metadata.FileName,
				metadata.Format,
			),
		}
	}

	sourceRelPath, err := paths.NewRelPath(metadata.FileName)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error building path for %q: %v\n",
			metadata.FileName,
			err,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message:  fmt.Sprintf("source path error: %v", err),
		}
	}

	sourceAbsPath, err := paths.JoinAbs(p.cfg.InDir, sourceRelPath)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error building path for %q: %v\n",
			metadata.FileName,
			err,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message:  fmt.Sprintf("source path error: %v", err),
		}
	}

	sourceHash, err := hashFile(sourceAbsPath)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error hashing %q: %v\n",
			metadata.FileName,
			err,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message:  fmt.Sprintf("file hashing error: %v", err),
		}
	}

	if existingImgDir, ok := imageDirsByHash[sourceHash]; ok {
		fmt.Fprintf(
			p.progressReporter,
			"Skipping %q: duplicate of image in %q\n",
			metadata.FileName,
			existingImgDir,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message:  fmt.Sprintf("skipping duplicate of image in %q", existingImgDir),
		}
	}

	imageProcessed, err := p.processFile(metadata, sourceHash, sourceAbsPath)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error processing %q: %v\n",
			metadata.FileName,
			err,
		)

		return image.Processed{}, &fileProblem{
			fileName: metadata.FileName,
			message:  fmt.Sprintf("file processing error: %v", err),
		}
	}

	imageDirsByHash[sourceHash] = imageProcessed.DirRelPath

	fmt.Fprintf(
		p.progressReporter,
		"Processed %q:\n\t%d variants generated in %q\n",
		metadata.FileName,
		len(imageProcessed.Variants),
		imageProcessed.DirRelPath,
	)

	return imageProcessed, nil
}

func (p *processor) recordMetadataProblems(problems []metadata.Problem) []fileProblem {
	var fileProblems []fileProblem

	for _, problem := range problems {
		fmt.Fprintf(
			p.progressReporter,
			"\t%q: %s\n",
			problem.FileName,
			problem.Message,
		)

		fileProblems = append(fileProblems, fileProblem{
			fileName: problem.FileName,
			message:  problem.Message,
		})
	}

	return fileProblems
}

func (p *processor) readIncomingMetadata() (metadata.Result, error) {
	metadataResult, err := p.metadataReader.Read(p.cfg.InDir)
	if err != nil {
		return metadata.Result{}, err
	}

	fmt.Fprintf(
		p.progressReporter,
		"Read metadata from %d file(s) with %d error(s)\n",
		len(metadataResult.Metadata)+len(metadataResult.FileProblems),
		len(metadataResult.FileProblems),
	)

	return metadataResult, nil
}

func loadExistingIndex(outDir paths.AbsPath) (index.Index, map[string]paths.RelPath, error) {
	imageIndex, err := index.ReadDir(outDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			imageIndex = index.Index{Images: []index.Entry{}}
		} else {
			return index.Index{}, nil, err
		}
	}

	imageDirsByHash, err := imageIndex.ImageDirsBySHA256()
	if err != nil {
		return index.Index{}, nil, err
	}

	return imageIndex, imageDirsByHash, nil
}

func deleteProcessedImageDirs(outRoot paths.AbsPath, images []image.Processed) error {
	var cleanupErr error

	for i, img := range images {
		imgDirAbsPath, err := paths.JoinAbs(outRoot, img.DirRelPath)
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("building cleanup path for image %d: %w", i, err))
			continue
		}

		if err := deleteRemnants(imgDirAbsPath); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("deleting image directory %q: %w", imgDirAbsPath, err))
		}
	}

	return cleanupErr
}

func cleanupImageDirOnError(imgDir paths.AbsPath, originalErr error) error {
	cleanupErr := deleteRemnants(imgDir)
	if cleanupErr != nil {
		return errors.Join(
			originalErr,
			fmt.Errorf("cleaning up image directory %q: %w", imgDir, cleanupErr),
		)
	}

	return originalErr
}

func (p *processor) processFile(metadata image.Metadata, sourceHash string, sourceAbsPath paths.AbsPath) (image.Processed, error) {
	processedAt := time.Now().UTC()
	dirDate, err := dirDate(p.cfg.DirDate, metadata.CapturedAt, processedAt)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirRelPath, err := imgDirRelPath(dirDate)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirAbsPath, err := paths.JoinAbs(p.cfg.OutDir, imgDirRelPath)
	if err != nil {
		return image.Processed{}, err
	}

	if err := os.MkdirAll(imgDirAbsPath.String(), 0755); err != nil {
		return image.Processed{}, fmt.Errorf("unable to make image directory %s: %w", imgDirAbsPath, err)
	}

	sourceDestRelPath, err := paths.NewRelPath("orig.jpg")
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(imgDirAbsPath, err)
	}
	sourceDest, err := paths.JoinAbs(imgDirAbsPath, sourceDestRelPath)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(imgDirAbsPath, err)
	}
	if err := copyFile(sourceAbsPath, sourceDest); err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to copy source %s to %s: %w", sourceAbsPath, imgDirAbsPath, err),
		)
	}

	sourceFile := image.Source{
		Hash:   sourceHash,
		Width:  metadata.Width,
		Height: metadata.Height,
	}

	processedImg := image.Processed{
		Source:      sourceFile,
		DirRelPath:  imgDirRelPath,
		Title:       metadata.Title,
		Description: metadata.Description,
		CapturedAt:  metadata.CapturedAt,
		ProcessedAt: image.FormatProcessedAt(processedAt),
	}

	specs, err := variantSpecs(imgDirAbsPath, processedImg)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(imgDirAbsPath, err)
	}
	result, err := p.variantGenerator.Generate(sourceAbsPath, specs)

	if len(result.Generated) == 0 {
		if err == nil {
			err = fmt.Errorf("%d variants were attempted, and no errors were reported", len(specs))
		}
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("no variants were generated: %w", err),
		)
	}

	processedImg.Variants, err = identifyVariants(result.Generated)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to identify generated variants: %w", err),
		)
	}

	mani, err := manifest.FromProcessed(processedImg)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to write manifest: %w", err),
		)
	}
	manifestPath, err := manifest.ManifestPath(imgDirAbsPath)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to write manifest: %w", err),
		)
	}
	if err := manifest.WriteFile(manifestPath, mani); err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to write manifest: %w", err),
		)
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

func dirDate(dirDate DirDate, capturedAt string, processedAt time.Time) (time.Time, error) {
	switch dirDate {
	case DirDateProcessed:
		return processedAt, nil
	case DirDateCaptured:
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

func variantSpecs(imgDir paths.AbsPath, img image.Processed) ([]variants.Spec, error) {
	var desiredWidths = []int{400, 800, 1200, 1600}
	var desiredExts = []string{".jpg", ".avif"}

	// widths generated are <= the source's width
	widths := variantWidths(img.Source.Width, desiredWidths)
	specs := make([]variants.Spec, 0, len(widths)*len(desiredExts))

	for _, ext := range desiredExts {
		for _, width := range widths {
			filename := filename(width, ext)
			variantRelPath, err := paths.NewRelPath(filename)
			if err != nil {
				return nil, err
			}
			outPath, err := paths.JoinAbs(imgDir, variantRelPath)
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
