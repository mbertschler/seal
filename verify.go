package seal

import (
	"log"

	"github.com/pkg/errors"
)

var PrintVerify = false

// VerifyPath checks all files and directories against the
// seal JSON files by comparing metadata and hashing file contents.
func VerifyPath(dirPath string) error {
	if PrintVerify {
		log.Println("verifying", dirPath)
	}

	dirs, err := indexDirectories(dirPath)
	if err != nil {
		return errors.Wrap(err, "indexDirectories")
	}

	checkHash := false
	for _, dir := range dirs {
		if PrintVerify {
			log.Println("quick checking", dir.path)
		}
		verifyDir(dir.path, checkHash)
	}

	checkHash = true
	for _, dir := range dirs {
		if PrintVerify {
			log.Println("hashing", dir.path)
		}
		verifyDir(dir.path, checkHash)
	}
	return nil
}

// verifyDir diffs the current contents of a directory
// against the stored seal, with or without hashing.
func verifyDir(dirPath string, checkHash bool) error {
	currentSeal, err := sealDir(dirPath, checkHash)
	if err != nil {
		return errors.Wrap(err, "sealDir")
	}

	loadedSeal, err := loadSeal(dirPath)
	if err != nil {
		return errors.Wrap(err, "loadSeal")
	}

	diff := DiffSeals(loadedSeal, currentSeal, checkHash)
	diff.PrintDifferences()
	return nil
}
