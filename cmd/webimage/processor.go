package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"tetrahemihexahedron/webgallery/internal/image"
	"tetrahemihexahedron/webgallery/internal/index"
	"tetrahemihexahedron/webgallery/internal/manifest"
	"tetrahemihexahedron/webgallery/internal/metadata"
	"tetrahemihexahedron/webgallery/internal/paths"
	"tetrahemihexahedron/webgallery/internal/variants"
)

type metadataReader func(context.Context, paths.AbsPath) (metadata.Result, error)

type variantGenerator func(context.Context, variants.Request) (variants.Result, error)

type processResult struct {
	images   []image.Processed
	problems []imageProblem
}

type imageProblem struct {
	fileName string
	message  string
}

type metadataEntryResult struct {
	processed image.Processed
	problem   *imageProblem
}

type sourceImage struct {
	metadata metadata.File
	path     paths.AbsPath
	sha256   string
}

type processorOptions struct {
	inDir   paths.AbsPath
	outDir  paths.AbsPath
	dirDate dirDateSource
}

type imageProcessor struct {
	options          processorOptions
	metadataReader   metadataReader
	variantGenerator variantGenerator
	progressReporter io.Writer
}

func (p *imageProcessor) processIncomingDir(ctx context.Context) (res processResult, retErr error) {
	defer func() {
		if retErr == nil {
			return
		}
		if cleanupErr := deleteProcessedImageDirs(p.options.outDir, res.images); cleanupErr != nil {
			retErr = errors.Join(retErr, cleanupErr)
		}
		res = processResult{}
	}()

	inDirAbsPath := p.options.inDir.String()

	fmt.Fprintf(p.progressReporter, "Processing image files in %q\n", inDirAbsPath)

	imageIndex, imageDirsByHash, err := loadExistingIndex(p.options.outDir)
	if err != nil {
		return processResult{}, err
	}

	metadataResult, err := p.readIncomingMetadata(ctx)
	if err != nil {
		return processResult{}, err
	}

	res = processResult{
		problems: p.recordMetadataProblems(metadataResult.FileProblems),
	}

	fmt.Fprint(p.progressReporter, "\n----------------\n")

	for _, meta := range metadataResult.Metadata {
		entry, err := p.processMetadataEntry(ctx, meta, imageDirsByHash)
		if err != nil {
			return res, err
		}
		if entry.problem != nil {
			res.problems = append(res.problems, *entry.problem)
			continue
		}
		res.images = append(res.images, entry.processed)
	}

	// Check for cancellation after the final metadata entry was processed
	if err := ctx.Err(); err != nil {
		return res, err
	}
	if err := p.writeUpdatedIndex(&imageIndex, res.images); err != nil {
		return res, err
	}

	return res, nil
}

func (p *imageProcessor) writeUpdatedIndex(imageIndex *index.Index, images []image.Processed) error {
	if err := imageIndex.Update(p.options.outDir, images); err != nil {
		return fmt.Errorf("updating index: %w", err)
	}
	return nil
}

func (p *imageProcessor) processMetadataEntry(ctx context.Context, meta metadata.File, imageDirsByHash map[string]paths.RelPath) (metadataEntryResult, error) {
	if err := ctx.Err(); err != nil {
		return metadataEntryResult{}, err
	}

	if image.ParseFormat(meta.Format) != image.FormatJPEG {
		fmt.Fprintf(
			p.progressReporter,
			"Skipping %q: format is %s, not JPEG\n",
			meta.FileName,
			meta.Format,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message: fmt.Sprintf(
				"skipping file %q: format is %s, not JPEG",
				meta.FileName,
				meta.Format,
			),
		}}, nil
	}

	if err := validateJPEGMetadata(meta, p.options.dirDate); err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error validating metadata for %q: %v\n",
			meta.FileName,
			err,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message:  fmt.Sprintf("metadata validation error: %v", err),
		}}, nil
	}

	sourceAbsPath, err := paths.JoinAbs(p.options.inDir, meta.FileName)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error building path for %q: %v\n",
			meta.FileName,
			err,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message:  fmt.Sprintf("source path error: %v", err),
		}}, nil
	}

	sourceHash, err := fileSHA256(sourceAbsPath)
	if err != nil {
		fmt.Fprintf(
			p.progressReporter,
			"Error hashing %q: %v\n",
			meta.FileName,
			err,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message:  fmt.Sprintf("file hashing error: %v", err),
		}}, nil
	}

	source := sourceImage{
		metadata: meta,
		path:     sourceAbsPath,
		sha256:   sourceHash,
	}

	if existingImgDir, ok := imageDirsByHash[source.sha256]; ok {
		fmt.Fprintf(
			p.progressReporter,
			"Skipping %q: duplicate of image in %q\n",
			meta.FileName,
			existingImgDir,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message:  fmt.Sprintf("skipping duplicate of image in %q", existingImgDir),
		}}, nil
	}

	processedImg, err := p.processImage(ctx, source)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return metadataEntryResult{}, err
		}

		fmt.Fprintf(
			p.progressReporter,
			"Error processing %q: %v\n",
			meta.FileName,
			err,
		)

		return metadataEntryResult{problem: &imageProblem{
			fileName: meta.FileName.String(),
			message:  fmt.Sprintf("file processing error: %v", err),
		}}, nil
	}

	imageDirsByHash[source.sha256] = processedImg.DirRelPath

	fmt.Fprintf(
		p.progressReporter,
		"Processed %q:\n\t%d variants generated in %q\n",
		meta.FileName,
		len(processedImg.Variants),
		processedImg.DirRelPath,
	)

	return metadataEntryResult{processed: processedImg}, nil
}

func validateJPEGMetadata(meta metadata.File, dirDate dirDateSource) error {
	if meta.Width <= 0 {
		return fmt.Errorf("width must be positive, got %d", meta.Width)
	}
	if meta.Height <= 0 {
		return fmt.Errorf("height must be positive, got %d", meta.Height)
	}
	if meta.CapturedAt == "" {
		if dirDate == dirDateCaptured {
			return errors.New("capturedAt is required with --dir-date=captured")
		}
		return nil
	}

	if _, err := image.ParseCapturedAt(meta.CapturedAt); err != nil {
		return fmt.Errorf("invalid capturedAt: %w", err)
	}

	return nil
}

func (p *imageProcessor) recordMetadataProblems(problems []metadata.Problem) []imageProblem {
	var imageProblems []imageProblem

	for _, problem := range problems {
		fmt.Fprintf(
			p.progressReporter,
			"\t%q: %s\n",
			problem.FileName,
			problem.Message,
		)

		imageProblems = append(imageProblems, imageProblem{
			fileName: problem.FileName,
			message:  problem.Message,
		})
	}

	return imageProblems
}

func (p *imageProcessor) readIncomingMetadata(ctx context.Context) (metadata.Result, error) {
	metadataResult, err := p.metadataReader(ctx, p.options.inDir)
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

func (p *imageProcessor) processImage(ctx context.Context, source sourceImage) (image.Processed, error) {
	processedAt := time.Now().UTC()
	dirDate, err := dirDate(p.options.dirDate, source.metadata.CapturedAt, processedAt)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirRelPath, err := newImageDirRelPath(dirDate)
	if err != nil {
		return image.Processed{}, err
	}
	imgDirAbsPath, err := createImageDir(p.options.outDir, imgDirRelPath)
	if err != nil {
		return image.Processed{}, err
	}

	sourceDestRelPath, err := paths.NewRelPath("orig.jpg")
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(imgDirAbsPath, err)
	}
	sourceDest, err := paths.JoinAbs(imgDirAbsPath, sourceDestRelPath)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(imgDirAbsPath, err)
	}
	if err := copyFile(source.path, sourceDest); err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to copy source %s to %s: %w", source.path, imgDirAbsPath, err),
		)
	}

	sourceFile := image.Source{
		Hash:   source.sha256,
		Width:  source.metadata.Width,
		Height: source.metadata.Height,
	}

	processedImg := image.Processed{
		Source:      sourceFile,
		DirRelPath:  imgDirRelPath,
		Title:       source.metadata.Title,
		Description: source.metadata.Description,
		CapturedAt:  source.metadata.CapturedAt,
		ProcessedAt: image.FormatProcessedAt(processedAt),
	}

	desiredWidths := []int{400, 800, 1200, 1600}
	request := variants.Request{
		SourcePath: source.path,
		OutputDir:  imgDirAbsPath,
		Widths:     variantWidths(source.metadata.Width, desiredWidths),
		Formats:    []image.Format{image.FormatJPEG, image.FormatAVIF},
	}
	result, err := p.variantGenerator(ctx, request)
	if err != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("unable to generate variants: %w", err),
		)
	}

	if variantErr := result.Err(); variantErr != nil {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			fmt.Errorf("variant generation incomplete: %d generated, %d failed: %w", len(result.Generated), len(result.Failed), variantErr),
		)
	}

	processedImg.Variants = result.Generated

	hasJPEG := false
	for _, variant := range processedImg.Variants {
		if variant.Format == image.FormatJPEG {
			hasJPEG = true
			break
		}
	}
	if !hasJPEG {
		return image.Processed{}, cleanupImageDirOnError(
			imgDirAbsPath,
			errors.New("no JPEG fallback was generated"),
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
