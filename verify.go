package seal

import (
	"log"

	"github.com/pkg/errors"
)

var PrintVerify = false

// VerifyPath checks all files and directories against the
// seal JSON files by comparing metadata and hashing file contents.
func VerifyPath(dirPath string, printDifferences bool) ([]*dir, error) {
	if PrintVerify {
		log.Println("indexing", dirPath)
	}
	dirs, err := indexDirectories(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "indexDirectories")
	}

	checkHash := false
	for _, dir := range dirs {
		if PrintVerify {
			log.Println("quick checking", dir.path)
		}
		diff, err := verifyDir(dir.path, checkHash)
		if err != nil {
			return nil, errors.Wrapf(err, "quick checking %q", dir.path)
		}
		if printDifferences {
			diff.PrintDifferences()
		}
		dir.quick = diff
	}

	checkHash = true
	for _, dir := range dirs {
		if PrintVerify {
			log.Println("hashing", dir.path)
		}
		diff, err := verifyDir(dir.path, checkHash)
		if err != nil {
			return nil, errors.Wrapf(err, "hashing %q", dir.path)
		}
		if printDifferences {
			diff.PrintDifferences()
		}
		dir.hash = diff
	}
	return dirs, nil
}

// verifyDir diffs the current contents of a directory
// against the stored seal, with or without hashing.
func verifyDir(dirPath string, checkHash bool) (*Diff, error) {
	currentSeal, err := sealDir(dirPath, checkHash)
	if err != nil {
		return nil, errors.Wrap(err, "sealDir")
	}

	loadedSeal, err := loadSeal(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "loadSeal")
	}

	diff := DiffSeals(loadedSeal, currentSeal, checkHash)
	return diff, nil
}
