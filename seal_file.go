package seal

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pkg/errors"
)

// SealFile is the filepath that is used in every sealed directory.
const SealFile = "_seal.json"

// DirSeal represents a complete seal of the directory
// including all files and subdirectories.
//
// The SHA256 is calculated by sorting the files by name
// and appending all sizes as 8 bytes in big endian format
// as well as the raw bytes of the files SHA256 hash.
type DirSeal struct {
	Name      string
	TotalSize int64
	Modified  time.Time
	Sealed    time.Time
	Verified  time.Time
	SHA256    []byte
	Files     []*FileSeal
}

// FileSeal represents one file inside a directory.
// If IsDir is true, the fields are populated from the
// seal file of the subdirectory with this name.
//
// The SHA256 is calculated from the contents of the file.
type FileSeal struct {
	Name     string
	IsDir    bool `json:",omitempty"`
	Size     int64
	Modified time.Time
	Sealed   time.Time
	Verified time.Time
	SHA256   []byte
}

// WriteFile writes the seal to the directory in JSON format.
func (d *DirSeal) WriteFile(dir string) error {
	d.sort()
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

// sort sorts the file array by names.
func (d *DirSeal) sort() {
	sort.Slice(d.Files, func(i, j int) bool {
		return d.Files[i].Name < d.Files[j].Name
	})
}

// hash calculates the SHA256 hash of the whole directory seal.
func (d *DirSeal) hash() error {
	d.sort()

	dirHash := sha256.New()
	for _, file := range d.Files {
		// sum the file size or total subdirectory size
		// converted into 8 bytes in big endian order
		size := make([]byte, 8)
		binary.BigEndian.PutUint64(size, uint64(file.Size))
		dirHash.Write(size)

		// sum the file hash
		_, err := dirHash.Write(file.SHA256)
		if err != nil {
			return errors.Wrap(err, "WriteHash")
		}
	}

	d.SHA256 = dirHash.Sum(nil)
	return nil
}
