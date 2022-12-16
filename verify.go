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
func VerifyPath(dirPath string, printDifferences bool) ([]Dir, error) {
	if PrintVerify {
		log.Println("indexing", dirPath)
	}
	loadSeals := false
	dirs, err := indexDirectories(dirPath, loadSeals)
	if err != nil {
		return nil, errors.Wrap(err, "indexDirectories")
	}

	dirsCount = len(dirs)
	verifyMode = "metadata"

	if PrintVerify {
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
			log.Println("quick checking", dir.Path)
		}
		diff, err := verifyDir(dir.Path, checkHash)
		if err != nil {
			return nil, errors.Wrapf(err, "quick checking %q", dir.Path)
		}
		if printDifferences {
			diff.PrintDifferences()
		}
		dir.QuickDiff = diff
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
			log.Println("hashing", dir.Path)
		}
		diff, err := verifyDir(dir.Path, checkHash)
		if err != nil {
			return nil, errors.Wrapf(err, "hashing %q", dir.Path)
		}
		if printDifferences {
			diff.PrintDifferences()
		}
		dir.HashDiff = diff
		sealingMeta.Lock()
		dirsDone++
		sealingMeta.Unlock()
	}

	if len(nonRegularFiles) > 0 {
		log.Println("skipped non regular files:")
		for mode, count := range nonRegularFiles {
			log.Println(mode.String(), count)
		}
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
