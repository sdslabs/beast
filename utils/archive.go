package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Compression int

const (
	Gzip Compression = 1
)

// Tar the provided context directory into the destination directory, additionalCtx is the context
// of one or more file which is to be added to the tar file
//
// In this function the contextDir is the absolute path of the directory to be compressed
// to the tar archive.
// DestincationDir is the directory to put the obtained tar file into.
func Tar(contextDir string, compression Compression, destinationDir string, additionalCtx map[string]string, subDirToSkip []string) error {
	e := ValidateDirExists(contextDir)
	if e != nil {
		return e
	}

	if compression != Gzip {
		return errors.New("only Gzipped compression is available")
	}

	outFile := fmt.Sprintf("%s.tar.gz", filepath.Base(contextDir))
	target := filepath.Join(destinationDir, outFile)

	err := ValidateFileExists(target)
	if err == nil {
		log.Warnf("The tar target you are trying to create already exists(%s), overriding", target)
		remErr := os.Remove(target)
		if remErr != nil {
			return errors.New("error while removing existing tar")
		}
	}

	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("error while creating tar :: %s", target)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(target)
		}
	}()
	defer targetFile.Close()

	// Create a Gzipped tar file writer from the file we just opened
	var tarFile io.WriteCloser = targetFile
	fileWriter := gzip.NewWriter(tarFile)
	tarFileWriter := tar.NewWriter(fileWriter)
	defer fileWriter.Close()
	defer tarFileWriter.Close()

	dirToSkip := func(dir string) bool {
		for _, d := range subDirToSkip {
			if dir == d {
				return true
			}
		}
		return false
	}

	baseDir := filepath.Base(contextDir)
	err = filepath.Walk(contextDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && dirToSkip(path) {
				log.Debugf("skipping a dir without errors: %s", info.Name())
				return filepath.SkipDir
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported file type in archive context: %s", path)
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(strings.TrimPrefix(path, contextDir))
			}

			if err := tarFileWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			curFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer curFile.Close()

			_, err = io.Copy(tarFileWriter, curFile)
			return err
		})

	// Add additional file to the tar.
	for fileName, filePath := range additionalCtx {
		fileInfo, err := CheckPathValid(filePath)
		if err != nil || !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("invalid additional archive file %s", filePath)
		}

		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return fmt.Errorf("create archive header for %s: %w", filePath, err)
		}

		header.Name = filepath.Join(fileName)

		if err := tarFileWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("write archive header for %s: %w", filePath, err)
		}

		curFile, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open additional archive file %s: %w", filePath, err)
		}

		_, err = io.Copy(tarFileWriter, curFile)
		closeErr := curFile.Close()
		if err != nil {
			return fmt.Errorf("write additional archive file %s: %w", filePath, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close additional archive file %s: %w", filePath, closeErr)
		}
	}

	if err != nil {
		log.Errorf("Error while creating tar for directory : %s", contextDir)
		log.Errorf("Removing corrupted tar which could not be created.")

		remErr := os.Remove(target)
		if remErr != nil {
			log.Errorf("Error while removing the corrupted tar file")
		}

		return fmt.Errorf("error while creating Tar :: %s", err)
	}
	if err := tarFileWriter.Close(); err != nil {
		return fmt.Errorf("finalize tar archive: %w", err)
	}
	if err := fileWriter.Close(); err != nil {
		return fmt.Errorf("finalize gzip archive: %w", err)
	}
	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	complete = true

	return nil
}
