package seal

import (
	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/bbolt"
)

type StorageType string

const IndexBoltDB StorageType = "boltdb"

var (
	pathsBucket  = []byte("paths")
	hashesBucket = []byte("hashes")
)

type IndexStorage interface {
	AddDir(dir *Dir, basePath string) error
	Close() error
}

type BoltIndex struct {
	db *bbolt.DB
}

func OpenBoltDB(indexPath string) (*BoltIndex, error) {
	db, err := bbolt.Open(indexPath, 0644, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "bbolt.Open")
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(pathsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(hashesBucket)
		if err != nil {
			return err
		}
		return nil
	})
	return &BoltIndex{db: db}, errors.Wrap(err, "setup db")
}

func (i *BoltIndex) Close() error {
	return i.db.Close()
}

type StoredSeal struct {
	Path string
	Dir  *DirSeal
	File *FileSeal
}

func (i *BoltIndex) AddDir(dir *Dir, basePath string) error {
	path, err := filepath.Rel(basePath, dir.Path)
	if err != nil {
		return errors.Wrap(err, "filepath.Rel")
	}
	toStore := []*StoredSeal{{
		Path: path,
		Dir:  dir.Seal,
	}}
	for _, file := range dir.Seal.Files {
		if file.IsDir {
			continue
		}
		toStore = append(toStore, &StoredSeal{
			Path: filepath.Join(path, file.Name),
			File: file,
		})
	}
	return i.db.Update(func(tx *bbolt.Tx) error {
		hashes := tx.Bucket(hashesBucket)
		paths := tx.Bucket(pathsBucket)

		for _, s := range toStore {
			var hash []byte
			if s.Dir != nil {
				hash = s.Dir.SHA256
			} else {
				hash = s.File.SHA256
			}
			buf, err := json.Marshal(s)
			if err != nil {
				return errors.Wrap(err, "json.Marshal")
			}
			err = hashes.Put(hash, buf)
			if err != nil {
				return errors.Wrap(err, "hashes.Put")
			}
			err = paths.Put([]byte(s.Path), hash)
			if err != nil {
				return errors.Wrap(err, "paths.Put")
			}
			putOps += 2

		}
		return nil
	})
}

var putOps int

var PrintDirsToIndex = true

func DirsToIndex(indexPath string, dirs []Dir, basePath string, t StorageType) error {
	var storage IndexStorage
	var err error
	switch t {
	case IndexBoltDB:
		storage, err = OpenBoltDB(indexPath)
		if err != nil {
			return errors.Wrap(err, "Open")
		}
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
