package utils

import (
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// Check if the path provided is a valid path, by calling stat on it
// if the path is invalid due to either non accesibility or existence
// an error is returned else FileInfo type is returned.
func checkPathValid(path string) (os.FileInfo, error) {
	// Check if the provided path exist
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Requested Path(%s) does not exist", path)
			return nil, errors.New("path does not exist")
		} else {
			log.Debugf("Requested Path(%s) is not accessbile", path)
			return nil, errors.New("not accessible path")
		}
	}

	return pathInfo, nil
}

// Validates if the directory pointed by `dirPath` exists
// If the directory does not exist or is not accessible it
// will return an error. The path specified must also be a directory
// and not just a regular file.
func ValidateDirExists(dirPath string) error {
	dirPathInfo, err := checkPathValid(dirPath)
	if err != nil {
		return err
	}

	if !dirPathInfo.IsDir() {
		log.Warnf("%s is not a directory", dirPath)
		return fmt.Errorf("not a directory : %s", dirPath)
	}

	return nil
}

// ValidateFileExists Validates if the file pointed by `filePath` exists.
// If the file does not exist or is not accessible it
// will return an error. The path specified must also be a valid file
func ValidateFileExists(filePath string) error {
	filePathInfo, err := checkPathValid(filePath)
	if err != nil {
		return err
	}

	// Check if the path provided points to a file
	if !filePathInfo.Mode().IsRegular() {
		log.Warnf("%s is not a file", filePath)
		return errors.New("not a file")
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
			eMsg := fmt.Errorf("could not create directory : %s", dirPath)
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
		return fmt.Errorf("error while removing existing file : %s : %s", filePath, err)
	}

	return nil
}

func CreateFileIfNotExist(filePath string) error {
	err := ValidateFileExists(filePath)
	if err != nil {
		file, e := os.Create(filePath)
		if e != nil {
			eMsg := fmt.Errorf("could not create file : %s", filePath)
			return eMsg
		}
		defer file.Close()
	}

	return nil
}
