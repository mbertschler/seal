package seal

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

var PrintSealing = false

// SealPath calculates seals for the given path and all subdirectories
// and writes them into a seal JSON file per directory.
func SealPath(dirPath string) ([]*dir, error) {
	if PrintSealing {
		log.Println("sealing", dirPath)
	}

	dirs, err := indexDirectories(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "indexDirectories")
	}

	for _, dir := range dirs {
		if PrintSealing {
			log.Println("sealing", dir.path)
		}
		hash := true
		seal, err := sealDir(dir.path, hash)
		if err != nil {
			return nil, errors.Wrapf(err, "sealDir %q", dir.path)
		}

		dir.seal = seal

		err = seal.UpdateSeal(dir.path, PrintSealing)
		if err != nil {
			return nil, errors.Wrapf(err, "seal.UpdateSeal %q", dir.path)
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
			return seal, errors.Wrap(err, "addFileToSeal")
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

// addFileToSeal appends a FileSeal to the DirSeal.
func addFileToSeal(seal *DirSeal, dirPath string, file fs.DirEntry, hash bool) error {
	if file.Name() == SealFile {
		return nil
	}
	fullPath := filepath.Join(dirPath, file.Name())

	var f *FileSeal
	var err error
	if file.IsDir() {
		f, err = sealSubDir(fullPath)
		if err != nil {
			return errors.Wrap(err, "sealSubDir")
		}
	} else {
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
