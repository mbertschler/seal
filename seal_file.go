package seal

import (
	"crypto/sha256"
	"encoding/base64"
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
	SHA256    []byte
	Modified  time.Time
	Sealed    time.Time
	// Verified  time.Time
	Files []*FileSeal
}

// FileSeal represents one file inside a directory.
// If IsDir is true, the fields are populated from the
// seal file of the subdirectory with this name.
//
// The SHA256 is calculated from the contents of the file.
type FileSeal struct {
	OldVersion bool `json:",omitempty"`
	Deleted    bool `json:",omitempty"`

	Name     string
	IsDir    bool `json:",omitempty"`
	Size     int64
	SHA256   []byte
	Modified time.Time
	Sealed   time.Time
	// Verified   time.Time
}

func (f *FileSeal) exists() bool {
	return !f.Deleted && !f.OldVersion
}

// UpdateSeal writes the seal to the directory in JSON format,
// joining it with the files seals of an al existing file.
func (d *DirSeal) UpdateSeal(dirPath string) error {
	loaded, err := loadSeal(dirPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "loadSeal")
	}
	if loaded != nil {
		d.joinWithExisting(loaded)
	}

	d.sort()
	file, err := os.Create(path.Join(dirPath, SealFile))
	if err != nil {
		return errors.Wrap(err, "Create seal")
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	err = enc.Encode(d)
	return errors.Wrap(err, "Encode seal")
}

// joinWithExisting adds deleted and changed files of a
// previous seal to the current seal Files slice.
func (d *DirSeal) joinWithExisting(existing *DirSeal) {
	checkHash := true
	diff := DiffSeals(existing, d, checkHash)

	// keep old versions and deleted files in the seal
	for _, file := range existing.Files {
		if !file.exists() {
			d.Files = append(d.Files, file)
		}
	}

	for _, file := range diff.FilesMissing {
		file.Deleted = true
		d.Files = append(d.Files, file)
	}
	for _, fd := range diff.FilesChanged {
		fd.Want.OldVersion = true
		d.Files = append(d.Files, fd.Want)
	}
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
		if !file.exists() {
			continue
		}

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

// Base64 encodes a byte slice to a base64 string.
func Base64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
