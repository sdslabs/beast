package utils

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Check if the path provided is a valid path, by calling stat on it
// if the path is invalid due to either non accesibility or existence
// an error is returned else FileInfo type is returned.
func CheckPathValid(path string) (os.FileInfo, error) {
	// Check if the provided path exist
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Requested Path(%s) does not exist", path)
			return nil, errors.New("Path does not exist")
		} else {
			log.Debugf("Requested Path(%s) is not accessbile", path)
			return nil, errors.New("Not accessible path.")
		}
	}

	return pathInfo, nil
}

// Validates if the directory pointed by `dirPath` exists
// If the directory does not exist or is not accessible it
// will return an error. The path specified must also be a directory
// and not just a regular file.
func ValidateDirExists(dirPath string) error {
	dirPathInfo, err := CheckPathValid(dirPath)
	if err != nil {
		return err
	}

	// Check if the path provided points to a directory
	if !dirPathInfo.IsDir() {
		log.Warnf("%s is not a directory", dirPath)
		return errors.New("Not a directory")
	}

	return nil
}

// Validates if the file pointed by `filePath` exists.
// If the file does not exist or is not accessible it
// will return an error. The path specified must also be a valid file
func ValidateFileExists(filePath string) error {
	filePathInfo, err := CheckPathValid(filePath)
	if err != nil {
		return err
	}

	// Check if the path provided points to a file
	if !filePathInfo.Mode().IsRegular() {
		log.Warnf("%s is not a file", filePath)
		return errors.New("Not a file")
	}

	return nil
}

// Create the directory sequence in dirPath if it does not exist
// if there was an error while creating the directory it returns the error
// else it returns nil indicating success
func CreateIfNotExistDir(dirPath string) error {
	err := ValidateDirExists(dirPath)
	if err != nil {
		if e := os.MkdirAll(dirPath, 0755); e != nil {
			eMsg := fmt.Errorf("Could not create directory : %s", dirPath)
			return eMsg
		}
	}

	return nil
}

func RemoveFileIfExists(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		return nil
	}

	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("Error while removing existing file : %s : %s", filePath, err)
	}

	return nil
}

func CreateFileIfNotExist(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		file, e := os.Create(filePath)
		defer file.Close()

		if e != nil {
			eMsg := fmt.Errorf("Could not create file : %s", filePath)
			return eMsg
		}
	}

	return nil
}

func RemoveDirRecursively(dirPath string) error {
	err := ValidateDirExists(dirPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("Error while removing directory %s : %s :: MAKE SURE TO CLEAN THE DIRECTORY YOURSELF", dirPath, err)
	}

	return nil
}

func CopyFile(src, dst string) error {
	err := ValidateFileExists(src)
	if err != nil {
		return err
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("Error while creating destination file : %s", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func CopyDirectory(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	fds, err := ioutil.ReadDir(src)

	if err != nil {
		return err
	}
	for _, fd := range fds {
		srcn := path.Join(src, fd.Name())
		dstn := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyDirectory(srcn, dstn); err != nil {
				return err
			}
		} else {
			if err = CopyFile(srcn, dstn); err != nil {
				return err
			}
		}
	}
	return nil
}

// This is very flexible and will not report any error even though it is not able to
// access the directory. It will return an empty list in such cases.  The caller must take
// care of this.
func GetAllDirectoriesName(dirPath string) []string {
	var directories []string

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			directories = append(directories, path)
		}

		return nil
	})

	return directories
}

// Returns a list of directory present in the provided directory
func GetDirsInDir(dirPath string) (error, []string) {
	var dirs []string
	err := ValidateDirExists(dirPath)
	if err != nil {
		return err, dirs
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("Error while reading directory with path %s : %s", dirPath, err), dirs
	}

	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return nil, dirs
}
