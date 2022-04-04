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

// SealPath calculates seals for the given path and all subdirectories
// and writes them into a seal JSON file per directory.
func SealPath(path string) error {
	log.Println("sealing", path)

	dirs, err := indexDirectories(path)
	if err != nil {
		return errors.Wrap(err, "directoriesToSeal")
	}

	for _, dir := range dirs {
		log.Println("sealing dir", dir.path)
		seal, err := sealDir(dir.path)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}

		err = seal.WriteFile(dir.path)
		if err != nil {
			return errors.Wrap(err, "seal.WriteDir")
		}
	}
	return nil
}

// sealDir turns all files and subdirectories into a DirSeal.
func sealDir(path string) (DirSeal, error) {
	var seal DirSeal

	// basic info from the directory itself
	info, err := os.Lstat(path)
	if err != nil {
		return seal, errors.Wrap(err, "Lstat")
	}
	seal.Name = info.Name()
	seal.Modified = info.ModTime()
	seal.Scanned = time.Now()

	// add information from all files and subdirectories to seal
	files, err := os.ReadDir(path)
	if err != nil {
		return seal, errors.Wrap(err, "ReadDir")
	}

	for _, file := range files {
		err = addFileToSeal(&seal, path, file)
		if err != nil {
			return seal, errors.Wrap(err, "addFileToSeal")
		}
	}

	err = seal.hash()
	if err != nil {
		return seal, errors.Wrap(err, "hash")
	}

	return seal, errors.Wrap(err, "WriteSeal")
}

// addFileToSeal appends a FileSeal to the DirSeal.
func addFileToSeal(seal *DirSeal, path string, file fs.DirEntry) error {
	if file.Name() == SealFile {
		return nil
	}
	fullPath := filepath.Join(path, file.Name())

	var f FileSeal
	var err error
	if file.IsDir() {
		f, err = sealSubDir(fullPath)
		if err != nil {
			return errors.Wrap(err, "sealSubDir")
		}
	} else {
		f, err = sealFile(fullPath)
		if err != nil {
			return errors.Wrap(err, "sealFile")
		}
	}

	seal.Files = append(seal.Files, f)
	seal.TotalSize += f.Size
	return nil
}

// sealFile turns a normal file into a FileSeal.
func sealFile(path string) (FileSeal, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return FileSeal{}, errors.Wrap(err, "Lstat")
	}

	if info.IsDir() {
		return FileSeal{}, errors.New("sealFile can't be used with directories")
	}

	seal := FileSeal{
		Name:     info.Name(),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
		Scanned:  time.Now(),
	}

	seal.SHA256, err = hashFile(path)
	return seal, errors.Wrap(err, "hashFile")
}

// hashFile hashes a normal file with SHA256.
func hashFile(path string) ([]byte, error) {
	fileHash := sha256.New()
	f, err := os.Open(path)
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
func sealSubDir(path string) (FileSeal, error) {
	var seal FileSeal
	f, err := os.Open(filepath.Join(path, SealFile))
	if err != nil {
		return seal, errors.Wrap(err, "Open")
	}
	defer f.Close()

	var dirSeal DirSeal
	err = json.NewDecoder(f).Decode(&dirSeal)
	if err != nil {
		return seal, errors.Wrap(err, "json.Decode")
	}

	seal = FileSeal{
		Name:     dirSeal.Name,
		IsDir:    true,
		Size:     dirSeal.TotalSize,
		Modified: dirSeal.Modified,
		Scanned:  dirSeal.Scanned,
		SHA256:   dirSeal.SHA256,
	}

	return seal, nil
}
