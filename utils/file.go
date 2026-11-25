package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// checkPathValid validates if the path provided exists and is accessible.
func checkPathValid(path string) (os.FileInfo, error) {
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Requested Path(%s) does not exist", path)
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		log.Debugf("Requested Path(%s) is not accessible: %v", path, err)
		return nil, fmt.Errorf("path is not accessible: %s: %w", path, err)
	}
	return pathInfo, nil
}

// ValidateDirExists validates if the directory pointed by dirPath exists and is a directory.
func ValidateDirExists(dirPath string) error {
	dirPathInfo, err := checkPathValid(dirPath)
	if err != nil {
		return err
	}
	if !dirPathInfo.IsDir() {
		log.Warnf("%s is not a directory", dirPath)
		return fmt.Errorf("not a directory: %s", dirPath)
	}
	return nil
}

// ValidateFileExists validates if the file pointed by filePath exists and is a regular file.
func ValidateFileExists(filePath string) error {
	filePathInfo, err := checkPathValid(filePath)
	if err != nil {
		return err
	}
	if !filePathInfo.Mode().IsRegular() {
		log.Warnf("%s is not a regular file", filePath)
		return fmt.Errorf("not a regular file: %s", filePath)
	}
	return nil
}

// ValidateFileReadable verifies that the file at filePath can be opened for reading.
func ValidateFileReadable(filePath string) error {
	if err := ValidateFileExists(filePath); err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file for reading: %s: %w", filePath, err)
	}
	_ = f.Close()
	return nil
}

// ValidateFileExecutable verifies that the file exists, is regular, and has executable permissions.
func ValidateFileExecutable(filePath string) error {
	info, err := checkPathValid(filePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("file is not executable: %s (permissions: %s)", filePath, info.Mode().Perm().String())
	}
	return nil
}

// WithRoot opens baseDir as an os.Root and runs fn safely inside it.
// All file operations executed through root are confined strictly beneath baseDir.
func WithRoot(baseDir string, fn func(root *os.Root) error) error {
	if strings.TrimSpace(baseDir) == "" {
		return errors.New("base directory cannot be empty")
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return fmt.Errorf("open root %q: %w", baseDir, err)
	}
	defer root.Close()

	return fn(root)
}

// SafeJoin securely joins a base directory with a relative path using WithRoot and os.Root.
// It verifies that:
// 1. relativePath is not empty, does not contain null bytes, and uses strictly forward slashes ('/').
// 2. Lexical cleaning does not escape baseDir (no '..', '.', or absolute paths).
// 3. Opening baseDir as an os.Root strictly confines file lookups beneath baseDir (preventing symlink and parent escapes).
func SafeJoin(baseDir, relativePath string) (string, error) {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		return "", errors.New("relative path cannot be empty")
	}
	if strings.ContainsRune(relativePath, 0) {
		return "", fmt.Errorf("relative path %q contains null byte", relativePath)
	}
	if strings.Contains(relativePath, `\`) {
		return "", fmt.Errorf("relative path %q must use forward slashes ('/'), backslashes are not allowed", relativePath)
	}

	cleanRel := filepath.Clean(relativePath)
	if cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanRel) || !filepath.IsLocal(cleanRel) {
		return "", fmt.Errorf("relative path %q traverses outside directory boundary", relativePath)
	}

	var fullPath string
	err := WithRoot(baseDir, func(root *os.Root) error {
		// Use root.Stat to verify that cleanRel does not escape the root boundary.
		// os.Root returns an error (e.g. "path escapes from parent") if cleanRel or
		// any symlink in its resolution chain points outside root.
		if _, err := root.Stat(cleanRel); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("path %q escapes root %q: %w", relativePath, baseDir, err)
			}
		}
		fullPath = filepath.Join(root.Name(), cleanRel)
		return nil
	})
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

// ValidateRelativePath validates that relPath is a safe, relative path within a directory
// without requiring it to exist on disk yet (useful for generated files).
func ValidateRelativePath(relPath string) error {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return errors.New("path cannot be empty")
	}
	if strings.ContainsRune(relPath, 0) {
		return fmt.Errorf("path %q contains null byte", relPath)
	}
	if strings.Contains(relPath, `\`) {
		return fmt.Errorf("path %q must use forward slashes ('/'), backslashes are not allowed", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) || !filepath.IsLocal(clean) {
		return fmt.Errorf("path %q must be a relative path inside the directory", relPath)
	}
	return nil
}

// ComputeFileSHA256 streams the file and computes its hex-encoded SHA256 checksum.
func ComputeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for SHA256 checksum: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetFileSize returns the file size in bytes for the specified file.
func GetFileSize(filePath string) (int64, error) {
	info, err := checkPathValid(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CreateIfNotExistDir creates the directory sequence in dirPath if it does not exist.
func CreateIfNotExistDir(dirPath string) error {
	err := ValidateDirExists(dirPath)
	if err != nil {
		if e := os.MkdirAll(dirPath, 0755); e != nil {
			return fmt.Errorf("could not create directory: %s: %w", dirPath, e)
		}
	}
	return nil
}

// CreateNewDir creates a new directory at dirPath and ensures that it does not already exist.
func CreateNewDir(dirPath string) error {
	if strings.TrimSpace(dirPath) == "" {
		return errors.New("directory path cannot be empty")
	}

	cleanPath := filepath.Clean(dirPath)
	if _, err := os.Stat(cleanPath); err == nil {
		return fmt.Errorf("directory already exists: %s: %w", cleanPath, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory path %s: %w", cleanPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return fmt.Errorf("could not create parent directories for: %s: %w", cleanPath, err)
	}

	if err := os.Mkdir(cleanPath, 0755); err != nil {
		return fmt.Errorf("could not create directory: %s: %w", cleanPath, err)
	}

	return nil
}

// RemoveFileIfExists removes the file at filePath if it exists.
func RemoveFileIfExists(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		return nil
	}
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("error while removing existing file: %s: %w", filePath, err)
	}
	return nil
}

// CreateFileIfNotExist creates an empty file at filePath if it does not exist.
func CreateFileIfNotExist(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		file, e := os.Create(filePath)
		if e != nil {
			return fmt.Errorf("could not create file: %s: %w", filePath, e)
		}
		defer file.Close()
	}
	return nil
}

// DeleteDir deletes the directory at dirPath and all its contents.
func DeleteDir(dirPath string) error {
	if strings.TrimSpace(dirPath) == "" {
		return errors.New("directory path cannot be empty")
	}

	cleanPath := filepath.Clean(dirPath)
	if cleanPath == "." || cleanPath == string(filepath.Separator) || cleanPath == filepath.VolumeName(cleanPath) || (filepath.VolumeName(cleanPath) != "" && cleanPath == filepath.VolumeName(cleanPath)+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove protected path: %s", dirPath)
	}

	if err := ValidateDirExists(cleanPath); err != nil {
		return err
	}

	if err := os.RemoveAll(cleanPath); err != nil {
		return fmt.Errorf("could not delete directory: %s: %w", cleanPath, err)
	}

	return nil
}

// RemoveDir is an alias for DeleteDir.
func RemoveDir(dirPath string) error {
	return DeleteDir(dirPath)
}
