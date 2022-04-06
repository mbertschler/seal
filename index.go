package seal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type dir struct {
	path  string
	depth int
	seal  *DirSeal
}

// indexDirectories returns all subdirectories with info about their depth.
// The deepest nested directories are sorted first.
func indexDirectories(dirPath string) ([]*dir, error) {
	info, err := os.Lstat(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "Lstat")
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dirPath)
	}

	out := []*dir{}
	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			path = filepath.Clean(path)
			parts := strings.Split(path, "/")
			out = append(out, &dir{path: path, depth: len(parts)})
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "WalkDir")
	}

	// deepest directories first, because we the seals for above directories
	sort.Slice(out, func(i, j int) bool {
		return out[i].depth > out[j].depth
	})
	return out, nil
}
