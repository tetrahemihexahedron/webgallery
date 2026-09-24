package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"

	"tetrahemihexahedron/webimage/internal/paths"
)

func fileSHA256(path paths.AbsPath) (string, error) {
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

func copyFile(source paths.AbsPath, dest paths.AbsPath) error {
	sourceFile, err := os.Open(source.String())
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dest.String(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(destFile, sourceFile)
	closeErr := destFile.Close()
	completionErr := errors.Join(copyErr, closeErr)
	if completionErr == nil {
		return nil
	}

	removeErr := os.Remove(dest.String())
	if errors.Is(removeErr, fs.ErrNotExist) {
		removeErr = nil
	}
	return errors.Join(completionErr, removeErr)
}
