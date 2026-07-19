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
	Gzip                     Compression = 1
	maxExtractedArchiveFiles             = 10000
	maxExtractedArchiveBytes             = int64(1 << 30)
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

			relativePath, err := filepath.Rel(contextDir, path)
			if err != nil {
				return fmt.Errorf("resolve archive path: %w", err)
			}
			header.Name = filepath.ToSlash(relativePath)

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

		cleanName := filepath.Clean(fileName)
		if cleanName == "." || filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid additional archive name %q", fileName)
		}
		header.Name = filepath.ToSlash(cleanName)

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

// ExtractTarGzip extracts a bounded archive into a new directory without
// following links or accepting paths outside that directory.
func ExtractTarGzip(archivePath, destination string) error {
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("archive destination already exists: %s", destination)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect archive destination: %w", err)
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("create archive destination: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer archiveFile.Close()
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	seen := make(map[string]struct{})
	var totalSize int64
	for count := 0; ; count++ {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if count >= maxExtractedArchiveFiles {
			return fmt.Errorf("archive contains more than %d entries", maxExtractedArchiveFiles)
		}
		name := filepath.Clean(filepath.FromSlash(header.Name))
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive path escapes destination: %q", header.Name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("archive contains duplicate path %q", header.Name)
		}
		seen[name] = struct{}{}
		if header.Size < 0 || header.Size > maxExtractedArchiveBytes-totalSize {
			return fmt.Errorf("archive expands beyond %d bytes", maxExtractedArchiveBytes)
		}
		totalSize += header.Size

		target := filepath.Join(destination, name)
		relative, err := filepath.Rel(destination, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive path escapes destination: %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if name == "." {
				continue
			}
			if err := os.Mkdir(target, header.FileInfo().Mode().Perm()); err != nil {
				return fmt.Errorf("create archive directory %q: %w", header.Name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return fmt.Errorf("create archive parent: %w", err)
			}
			file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, header.FileInfo().Mode().Perm())
			if err != nil {
				return fmt.Errorf("create archive file %q: %w", header.Name, err)
			}
			written, copyErr := io.Copy(file, io.LimitReader(reader, header.Size+1))
			closeErr := file.Close()
			if copyErr != nil {
				return fmt.Errorf("extract archive file %q: %w", header.Name, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close archive file %q: %w", header.Name, closeErr)
			}
			if written != header.Size {
				return fmt.Errorf("archive file %q size mismatch", header.Name)
			}
		default:
			return fmt.Errorf("archive contains unsupported entry %q", header.Name)
		}
	}
	complete = true
	return nil
}
