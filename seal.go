package seal

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const SealFile = "_seal.json"

// RootCmd is the what that should be executed by the seal command.
var RootCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal checks the integrity of your file archives",
	RunE:  runRootCmd,
}

func runRootCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("Hello, I'm seal! 🦭")
	if len(args) == 0 {
		return errors.New("need at least one path argument to seal")
	}
	for _, arg := range args {
		err := sealPath(arg)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}
	}
	return nil
}

func sealPath(path string) error {
	fmt.Println("sealing", path)
	dirs, err := directoriesToSeal(path)
	if err != nil {
		return errors.Wrap(err, "directoriesToSeal")
	}
	for _, dir := range dirs {
		fmt.Println("sealing dir", dir.path)
		err = sealDir(dir.path)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}
	}
	return nil
}

func sealDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return errors.Wrap(err, "Lstat")
	}
	files, err := os.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, "ReadDir")
	}

	seal := DirSeal{
		Name:     info.Name(),
		Modified: info.ModTime(),
		Scanned:  time.Now(),
	}

	for _, file := range files {
		if file.Name() == SealFile {
			continue
		}
		fullPath := filepath.Join(path, file.Name())
		var f FileSeal
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
	}

	seal.Sort()
	dirHash := sha256.New()
	for _, file := range seal.Files {
		size := make([]byte, 8)
		binary.BigEndian.PutUint64(size, uint64(file.Size))
		dirHash.Write(size)

		_, err = dirHash.Write(file.SHA256)
		if err != nil {
			return errors.Wrap(err, "WriteHash")
		}
	}
	seal.SHA256 = dirHash.Sum(nil)

	err = seal.WriteFile(path)
	return errors.Wrap(err, "WriteSeal")
}

func sealFile(path string) (FileSeal, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return FileSeal{}, errors.Wrap(err, "Lstat")
	}

	seal := FileSeal{
		Name:     info.Name(),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
		Scanned:  time.Now(),
	}

	if seal.IsDir {
		fmt.Println("sealFile for directories not implemented", path)
		return seal, nil
	}

	fileHash := sha256.New()
	f, err := os.Open(path)
	if err != nil {
		return seal, errors.Wrap(err, "Open")
	}
	defer f.Close()

	_, err = io.Copy(fileHash, f)
	if err != nil {
		return seal, errors.Wrap(err, "Copy")
	}
	seal.SHA256 = fileHash.Sum(nil)

	return seal, nil
}

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

func directoriesToSeal(path string) ([]dir, error) {
	out := []dir{}

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			path = filepath.Clean(path)
			parts := strings.Split(path, "/")
			out = append(out, dir{path: path, depth: len(parts)})
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "WalkDir")
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].depth > out[j].depth
	})
	return out, nil
}

type dir struct {
	path  string
	depth int
}
