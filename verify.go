package seal

import (
	"log"
	"time"

	"github.com/pkg/errors"
)

var (
	PrintVerify    = false
	PrintAllVerify = false
	verifyMode     = ""
)

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

	dirsCount = len(dirs)
	verifyMode = "metadata"

	if PrintSealing {
		tick := time.NewTicker(PrintInterval)
		defer tick.Stop()
		stop := make(chan bool)
		go func() {
			for {
				select {
				case <-tick.C:
					sealingMeta.Lock()
					log.Printf("%.1f%% done, %s %s", float64(dirsDone)/float64(dirsCount)*100, verifyMode, sealingFile)
					sealingMeta.Unlock()
				case <-stop:
					return
				}
			}
		}()
	}

	checkHash := false
	for _, dir := range dirs {
		if PrintAllVerify {
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
		sealingMeta.Lock()
		dirsDone++
		sealingMeta.Unlock()
	}

	sealingMeta.Lock()
	dirsDone = 0
	verifyMode = "hashing"
	sealingMeta.Unlock()

	checkHash = true
	for _, dir := range dirs {
		if PrintAllVerify {
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
		sealingMeta.Lock()
		dirsDone++
		sealingMeta.Unlock()
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
