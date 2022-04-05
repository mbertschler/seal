package seal

import (
	"log"

	"github.com/pkg/errors"
)

// VerifyPath checks all files and directories against the
// seal JSON files by hashing all the contents.
func VerifyPath(dirPath string) error {
	log.Println("verifying", dirPath)

	dirs, err := indexDirectories(dirPath)
	if err != nil {
		return errors.Wrap(err, "indexDirectories")
	}

	checkHash := false
	for _, dir := range dirs {
		log.Println("quick checking", dir.path)

		currentSeal, err := sealDir(dir.path, checkHash)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}

		loadedSeal, err := loadSeal(dir.path)
		if err != nil {
			return errors.Wrap(err, "loadSeal")
		}

		diff := DiffSeals(loadedSeal, currentSeal, checkHash)
		if !diff.Identical {
			log.Println("seals differ for", dir.path)
		}
	}
	return nil
}
