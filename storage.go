package seal

import (
	"log"
	"time"

	"github.com/pkg/errors"
)

var PrintDirsToIndex = true

const loadFromIndex = 10e3

type StorageType string

type IndexStorage interface {
	AddDir(dir *Dir, basePath string) error
	LoadAfterHash(hash []byte, count int) ([]StoredSeal, error)
	Close() error
}

func openStorage(t StorageType, path string) (IndexStorage, error) {
	switch t {
	case IndexBoltDB:
		storage, err := OpenBoltDB(path)
		return storage, errors.Wrap(err, "OpenBoltDB")
	case IndexSQLite:
		storage, err := OpenSqlite(path)
		return storage, errors.Wrap(err, "OpenSqlite")
	default:
		return nil, errors.Errorf("unknown storage type %q", t)
	}
}

func DirsToIndex(indexPath string, dirs []Dir, basePath string, t StorageType) error {
	storage, err := openStorage(t, indexPath)
	if err != nil {
		return errors.Wrap(err, "openStorage")
	}
	defer storage.Close()

	var tick *time.Ticker
	if PrintIndexProgress {
		tick = time.NewTicker(IndexProgressInterval)
		defer tick.Stop()
	}

	for i, dir := range dirs {
		err := storage.AddDir(&dir, basePath)
		if err != nil {
			return errors.Wrap(err, "AddDir")
		}
		if PrintIndexProgress {
			select {
			case <-tick.C:
				log.Printf("added %.1f%% to index %q", float64(i)/float64(len(dirs))*100, dir.Path)
			default:
			}
		}
	}
	return nil
}

type LoadedIndex struct {
	Dirs   []Dir
	ByHash map[string]*StoredSeal
	ByPath map[string]*StoredSeal
}

func LoadIndex(indexPath string, t StorageType) (*LoadedIndex, error) {
	storage, err := openStorage(t, indexPath)
	if err != nil {
		return nil, errors.Wrap(err, "openStorage")
	}
	defer storage.Close()

	var tick *time.Ticker
	if PrintIndexProgress {
		tick = time.NewTicker(IndexProgressInterval)
		defer tick.Stop()
	}

	var lastHash []byte

	out := &LoadedIndex{
		ByHash: map[string]*StoredSeal{},
		ByPath: map[string]*StoredSeal{},
	}
	hashes := 0
	for {
		stored, err := storage.LoadAfterHash(lastHash, loadFromIndex)
		hashes += len(stored)
		if err != nil {
			return nil, errors.Wrap(err, "LoadAfterHash")
		}
		if err == nil && len(stored) == 0 {
			break
		}

		for _, s := range stored {
			if s.Dir != nil && s.File != nil {
				return nil, errors.Errorf("both dir and file set for %q", s.Path)
			}
			if s.Dir != nil {
				out.Dirs = append(out.Dirs, Dir{
					Path: s.Path,
					//Depth?
					Seal: s.Dir,
				})
				out.ByHash[string(s.Dir.SHA256)] = &s
				out.ByPath[string(s.Path)] = &s
				lastHash = s.Dir.SHA256
			} else if s.File != nil {
				out.ByHash[string(s.File.SHA256)] = &s
				out.ByPath[string(s.Path)] = &s
				lastHash = s.File.SHA256
			} else {
				return nil, errors.Errorf("neither dir or file are set for %q", s.Path)
			}
		}

		if PrintIndexProgress {
			select {
			case <-tick.C:
				log.Printf("loaded %d dirs and %d hashes", len(out.Dirs), len(out.ByHash))
			default:
			}
		}
	}
	return out, nil
}
