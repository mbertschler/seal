package seal

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pkg/errors"
)

type DirSeal struct {
	Name      string
	TotalSize int64
	Modified  time.Time
	Scanned   time.Time
	SHA256    []byte
	Files     []FileSeal
}

type FileSeal struct {
	Name     string
	IsDir    bool `json:",omitempty"`
	Size     int64
	Modified time.Time
	Scanned  time.Time
	SHA256   []byte
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
