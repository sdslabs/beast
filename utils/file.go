package utils

import (
	"errors"
	"os"

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
			log.Errorf("Could not create directory : %s", dirPath)
			return e
		}
	}

	return nil
}

func CreateFileIfNotExist(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		if e := os.Create(filePath); e != nil {
			log.Errorf("Could not create file : %s", filePath)
			return e
		}
	}

	return nil
}
