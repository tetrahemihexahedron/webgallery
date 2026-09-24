package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"tetrahemihexahedron/webgallery/internal/image"
	"tetrahemihexahedron/webgallery/internal/paths"
)

func dirDate(dirDate dirDateSource, capturedAt string, processedAt time.Time) (time.Time, error) {
	switch dirDate {
	case dirDateProcessed:
		return processedAt, nil
	case dirDateCaptured:
		capturedDate, err := image.ParseCapturedAt(capturedAt)
		if err != nil {
			return time.Time{}, fmt.Errorf("--dir-date=captured requires a valid capturedAt: %w", err)
		}
		return capturedDate, nil
	default:
		return time.Time{}, fmt.Errorf("invalid directory date %q", dirDate)
	}
}

func newImageDirRelPath(date time.Time) (paths.RelPath, error) {
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

func createImageDir(outRoot paths.AbsPath, imgDirRelPath paths.RelPath) (paths.AbsPath, error) {
	imgDirAbsPath, err := paths.JoinAbs(outRoot, imgDirRelPath)
	if err != nil {
		return paths.AbsPath{}, err
	}

	parentDir := filepath.Dir(imgDirAbsPath.String())
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return paths.AbsPath{}, fmt.Errorf("unable to make image directory parents %q: %w", parentDir, err)
	}
	if err := os.Mkdir(imgDirAbsPath.String(), 0755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return paths.AbsPath{}, fmt.Errorf("image directory %q already exists: %w", imgDirAbsPath, err)
		}
		return paths.AbsPath{}, fmt.Errorf("unable to make image directory %q: %w", imgDirAbsPath, err)
	}

	return imgDirAbsPath, nil
}

func cleanupImageDirOnError(imgDir paths.AbsPath, originalErr error) error {
	cleanupErr := removeImageDir(imgDir)
	if cleanupErr != nil {
		return errors.Join(
			originalErr,
			fmt.Errorf("cleaning up image directory %q: %w", imgDir, cleanupErr),
		)
	}

	return originalErr
}

func deleteProcessedImageDirs(outRoot paths.AbsPath, images []image.Processed) error {
	var cleanupErr error

	for i, img := range images {
		imgDirAbsPath, err := paths.JoinAbs(outRoot, img.DirRelPath)
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("building cleanup path for image %d: %w", i, err))
			continue
		}

		if err := removeImageDir(imgDirAbsPath); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("deleting image directory %q: %w", imgDirAbsPath, err))
		}
	}

	return cleanupErr
}

func removeImageDir(dir paths.AbsPath) error {
	return os.RemoveAll(dir.String())
}
