package seal

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
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
		err := sealDir(arg)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}
	}
	return nil
}

func sealDir(path string) error {
	fmt.Println("sealing", path)
	dirs, err := directoriesToSeal(path)
	if err != nil {
		return errors.Wrap(err, "directoriesToSeal")
	}
	for _, dir := range dirs {
		fmt.Println("dir", dir)
	}
	return nil
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

type DirSeal struct {
	Name      string
	TotalSize int
	Modified  time.Time
	Scanned   time.Time
	SHA256    []byte
	Files     []FileSeal
}

func (d *DirSeal) Sort() {
	sort.Slice(d.Files, func(i, j int) bool {
		return d.Files[i].Name < d.Files[j].Name
	})
}

func (d *DirSeal) WriteFile(dir string) error {
	d.Sort()
	file, err := os.Create(path.Join(dir, SealFile))
	if err != nil {
		return errors.Wrap(err, "Create seal")
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	err = enc.Encode(d)
	return errors.Wrap(err, "Encode seal")
}

type FileSeal struct {
	Name     string
	IsDir    bool
	Size     int
	Modified time.Time
	Scanned  time.Time
	SHA256   []byte
}
