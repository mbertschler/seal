package seal

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type dir struct {
	path  string
	depth int
}

// indexDirectories returns all subdirectories with info about their depth.
// The deepest nested directories are sorted first.
func indexDirectories(path string) ([]dir, error) {
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

	// deepest directories first, because we the seals for above directories
	sort.Slice(out, func(i, j int) bool {
		return out[i].depth > out[j].depth
	})
	return out, nil
}
