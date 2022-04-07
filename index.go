package seal

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type dir struct {
	path  string
	depth int

	seal  *DirSeal
	quick *Diff
	hash  *Diff
}

var (
	PrintIndexProgress    = false
	IndexProgressInterval = 15 * time.Second
)

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

	var tick *time.Ticker
	if PrintIndexProgress {
		tick = time.NewTicker(IndexProgressInterval)
		defer tick.Stop()
	}

	out := []*dir{}
	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println(color.YellowString("can't index %q: %v", path, err))
			return nil
		}
		if d.IsDir() {
			path = filepath.Clean(path)
			parts := strings.Split(path, "/")
			out = append(out, &dir{path: path, depth: len(parts)})

			if PrintIndexProgress {
				select {
				case <-tick.C:
					log.Printf("indexing %q in depth %d", path, len(parts))
				default:
				}
			}
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
