package seal

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

var (
	PrintSealing    = false
	PrintAllSealing = false
	sealingMeta     sync.Mutex
	sealingFile     string
	dirsCount       int
	dirsDone        int

	filesToIgnore = map[string]bool{
		SealFile:    true,
		".DS_Store": true,
	}
)

// SealPath calculates seals for the given path and all subdirectories
// and writes them into a seal JSON file per directory.
func SealPath(dirPath string) ([]Dir, error) {
	if PrintSealing {
		log.Println("indexing", dirPath)
	}
	loadSeals := false
	dirs, err := indexDirectories(dirPath, loadSeals)
	if err != nil {
		return nil, errors.Wrap(err, "indexDirectories")
	}

	dirsCount = len(dirs)

	if PrintSealing {
		tick := time.NewTicker(PrintInterval)
		defer tick.Stop()
		stop := make(chan bool)
		go func() {
			for {
				select {
				case <-tick.C:
					sealingMeta.Lock()
					log.Printf("%.1f%% done, sealing %s", float64(dirsDone)/float64(dirsCount)*100, sealingFile)
					sealingMeta.Unlock()
				case <-stop:
					return
				}
			}
		}()
	}

	for _, dir := range dirs {
		if PrintAllSealing {
			log.Println("sealing", dir.Path)
		}
		hash := true
		seal, err := sealDir(dir.Path, hash)
		if err != nil {
			return nil, errors.Wrapf(err, "sealDir %q", dir.Path)
		}

		dir.Seal = seal

		err = seal.UpdateSeal(dir.Path, PrintSealing)
		if err != nil {
			log.Println(color.RedString("can't update seal: %v", err))
		}

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

// sealDir turns all files and subdirectories into a DirSeal.
func sealDir(dirPath string, hash bool) (*DirSeal, error) {
	// basic info from the directory itself
	info, err := os.Lstat(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "Lstat")
	}
	seal := &DirSeal{
		Name:     info.Name(),
		Modified: info.ModTime(),
		Sealed:   time.Now(),
	}

	// add information from all files and subdirectories to seal
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return seal, errors.Wrap(err, "ReadDir")
	}

	for _, file := range files {
		err = addFileToSeal(seal, dirPath, file, hash)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				log.Println(color.YellowString("file doesn't exist: %v", err))
			} else {
				log.Println(color.RedString("unexpected error in addFileToSeal: %v", err))
			}
		}
	}

	if hash {
		err = seal.hash()
		if err != nil {
			return seal, errors.Wrap(err, "hash")
		}
	}

	return seal, errors.Wrap(err, "WriteSeal")
}

var nonRegularFiles = map[os.FileMode]int{}

// addFileToSeal appends a FileSeal to the DirSeal.
func addFileToSeal(seal *DirSeal, dirPath string, file fs.DirEntry, hash bool) error {
	if filesToIgnore[file.Name()] {
		return nil
	}
	fullPath := filepath.Join(dirPath, file.Name())

	sealingMeta.Lock()
	sealingFile = fullPath
	sealingMeta.Unlock()

	var f *FileSeal
	var err error
	if file.IsDir() {
		f, err = sealSubDir(fullPath)
		if err != nil {
			return errors.Wrap(err, "sealSubDir")
		}
	} else {
		if !file.Type().IsRegular() {
			// log.Printf("not a regular file %s %q", file.Type().String(), fullPath)
			nonRegularFiles[file.Type()]++
			return nil
		}

		f, err = sealFile(fullPath, hash)
		if err != nil {
			return errors.Wrap(err, "sealFile")
		}
	}

	seal.Files = append(seal.Files, f)
	seal.TotalSize += f.Size
	return nil
}

// sealFile turns a normal file into a FileSeal.
func sealFile(filePath string, hash bool) (*FileSeal, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "Lstat")
	}

	if info.IsDir() {
		return nil, errors.New("sealFile can't be used with directories")
	}

	seal := &FileSeal{
		Name:     info.Name(),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
		Sealed:   time.Now(),
	}

	if !hash {
		return seal, nil
	}

	seal.SHA256, err = hashFile(filePath)
	return seal, errors.Wrap(err, "hashFile")
}

// hashFile hashes a normal file with SHA256.
func hashFile(filePath string) ([]byte, error) {
	fileHash := sha256.New()
	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "Open")
	}
	defer f.Close()

	_, err = io.Copy(fileHash, f)
	if err != nil {
		return nil, errors.Wrap(err, "Copy")
	}

	return fileHash.Sum(nil), nil
}

// sealSubDir turns the seal file of a subdirectory into a FileSeal.
func sealSubDir(dirPath string) (*FileSeal, error) {
	dirSeal, err := loadSeal(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "loadSeal")
	}

	seal := &FileSeal{
		Name:     dirSeal.Name,
		IsDir:    true,
		Size:     dirSeal.TotalSize,
		Modified: dirSeal.Modified,
		Sealed:   dirSeal.Sealed,
		SHA256:   dirSeal.SHA256,
	}

	return seal, nil
}

// loadSeal loads the seal file of a directory.
// Don't include the seal file itself in the path.
func loadSeal(dirPath string) (*DirSeal, error) {
	f, err := os.Open(filepath.Join(dirPath, SealFile))
	if err != nil {
		return nil, errors.Wrap(err, "Open")
	}
	defer f.Close()

	var dirSeal DirSeal
	err = json.NewDecoder(f).Decode(&dirSeal)
	if err != nil {
		return nil, errors.Wrap(err, "json.Decode")
	}
	return &dirSeal, nil
}
