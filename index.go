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

func IndexPath(path, indexFile string) error {
	log.Println("indexing", path)
	start := time.Now()
	dirs, err := indexDirectories(path, true)
	if err != nil {
		return errors.Wrap(err, "indexDirectories")
	}
	log.Println("loaded", len(dirs), "directories with seals in", time.Since(start))

	return DirsToIndex(indexFile, dirs, path)
}

// indexDirectories returns all subdirectories with info about their depth.
// The deepest nested directories are sorted first.
func indexDirectories(dirPath string, loadSeals bool) ([]*dir, error) {
	info, err := os.Lstat(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "Lstat")
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dirPath)
	}

	if !Before.IsZero() {
		loadSeals = true
	}

	var tick *time.Ticker
	if PrintIndexProgress {
		tick = time.NewTicker(IndexProgressInterval)
		defer tick.Stop()
	}

	skipped := 0
	out := []*dir{}
	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println(color.YellowString("can't index %q: %v", path, err))
			return nil
		}
		if d.IsDir() {
			var seal *DirSeal
			if loadSeals {
				seal, err = loadSeal(path)
			}
			if !Before.IsZero() {
				if err == nil && seal.Sealed.After(Before) {
					skipped++
					return fs.SkipDir
				}
			}

			path = filepath.Clean(path)
			parts := strings.Split(path, "/")
			out = append(out, &dir{path: path, depth: len(parts), seal: seal})

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

	if PrintIndexProgress {
		log.Println("indexed", len(out), "directories and skipped", skipped)
	}

	// deepest directories first, because we the seals for above directories
	sort.Slice(out, func(i, j int) bool {
		return out[i].depth > out[j].depth
	})
	return out, nil
}
