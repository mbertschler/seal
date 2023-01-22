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

type Dir struct {
	Path  string
	Depth int

	Seal *DirSeal

	QuickDiff *Diff
	HashDiff  *Diff
}

var (
	PrintIndexProgress    = false
	IndexProgressInterval = 15 * time.Second
)

func IndexPath(path, indexFile string, prefixes []string) error {
	log.Printf("indexing %q with prefixes %q", path, prefixes)
	start := time.Now()
	dirs, err := indexDirectories(path, true, prefixes)
	if err != nil {
		return errors.Wrap(err, "indexDirectories")
	}
	log.Println("loaded", len(dirs), "directories with seals in", time.Since(start))

	start = time.Now()
	err = DirsToIndex(indexFile, dirs, path, IndexSQLite)
	if err != nil {
		return errors.Wrap(err, "DirsToIndex")
	}
	log.Println("indexed", len(dirs), "directories in", time.Since(start))
	return nil
}

// indexDirectories returns all subdirectories with info about their depth.
// The deepest nested directories are sorted first.
func indexDirectories(dirPath string, loadSeals bool, prefixes []string) ([]Dir, error) {
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
	out := []Dir{}
	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println(color.YellowString("can't index %q: %v", path, err))
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return errors.Wrap(err, "filepath.Rel")
		}
		if !isInPrefixes(relPath, prefixes) {
			return fs.SkipDir
		}

		var seal *DirSeal
		if loadSeals {
			seal, err = loadSeal(path)
			if seal == nil {
				skipped++
				return fs.SkipDir
			}
		}
		if !Before.IsZero() {
			if err == nil && seal.Sealed.After(Before) {
				skipped++
				return fs.SkipDir
			}
		}

		path = filepath.Clean(path)
		parts := strings.Split(path, "/")
		out = append(out, Dir{Path: path, Depth: len(parts), Seal: seal})

		if PrintIndexProgress {
			select {
			case <-tick.C:
				log.Printf("indexing %q in depth %d", path, len(parts))
			default:
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
		return out[i].Depth > out[j].Depth
	})
	return out, nil
}

func isInPrefixes(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	if path == "." {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
