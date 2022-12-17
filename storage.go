package seal

import (
	"log"
	"time"

	"github.com/pkg/errors"
)

var PrintDirsToIndex = true

type StorageType string

type IndexStorage interface {
	AddDir(dir *Dir, basePath string) error
	Close() error
}

func DirsToIndex(indexPath string, dirs []Dir, basePath string, t StorageType) error {
	var storage IndexStorage
	var err error
	switch t {
	case IndexBoltDB:
		storage, err = OpenBoltDB(indexPath)
		if err != nil {
			return errors.Wrap(err, "Open")
		}
	case IndexSQLite:
		storage, err = OpenSqlite(indexPath)
		if err != nil {
			return errors.Wrap(err, "Open")
		}
	default:
		return errors.Errorf("unknown storage type %q", t)
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
