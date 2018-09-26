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
func Tar(contextDir string, compression Compression, destinationDir string, additionalCtx map[string]string) error {
	e := ValidateDirExists(contextDir)
	if e != nil {
		return e
	}

	if compression != Gzip {
		return errors.New("Only Gzipped compression is available")
	}

	outFile := fmt.Sprintf("%s.tar.gz", filepath.Base(contextDir))
	target := filepath.Join(destinationDir, outFile)

	err := ValidateFileExists(target)
	if err == nil {
		log.Warnf("The tar target you are trying to create already exists(%s), overriding", target)
		remErr := os.Remove(target)
		if remErr != nil {
			return errors.New("Error while removing existing tar")
		}
	}

	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("Error while creating tar :: %s", target)
	}
	defer targetFile.Close()

	// Create a Gzipped tar file writer from the file we just opened
	var tarFile io.WriteCloser = targetFile
	fileWriter := gzip.NewWriter(tarFile)
	tarFileWriter := tar.NewWriter(fileWriter)
	defer fileWriter.Close()
	defer tarFileWriter.Close()

	baseDir := filepath.Base(contextDir)
	err = filepath.Walk(contextDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, contextDir))
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
			log.Errorf("Cannot find a valid file for %s while creating tar... Continuing", filePath)
			continue
		}

		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			log.Errorf("Cannot create a valid tar header for %s while creating tar... Continuing", filePath)
			continue
		}

		header.Name = filepath.Join(baseDir, fileName)

		if err := tarFileWriter.WriteHeader(header); err != nil {
			log.Errorf("Cannot write tar header for %s while creating tar... Continuing", filePath)
			continue
		}

		curFile, _ := os.Open(filePath)

		_, err = io.Copy(tarFileWriter, curFile)
		if err != nil {
			log.Errorf("Cannot write file to tar %s... Continuing", filePath)
			continue
		}
		curFile.Close()
	}

	if err != nil {
		log.Errorf("Error while creating tar for directory : %s", contextDir)
		log.Errorf("Removing corrupted tar which could not be created.")

		remErr := os.Remove(target)
		if remErr != nil {
			log.Errorf("Error while removing the corrupted tar file")
		}

		return fmt.Errorf("Error while creating Tar :: %s", err)
	}

	return nil
}
