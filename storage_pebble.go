package seal

import (
	"encoding/json"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
)

const StorageTypePebble StorageType = "pebble"

var (
	pathsPrefix  = []byte("paths/")
	hashesPrefix = []byte("hashes/")
)

type PebbleIndex struct {
	db *pebble.DB
}

func OpenPebble(indexPath string) (*PebbleIndex, error) {
	db, err := pebble.Open(indexPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "pebble.Open")
	}
	return &PebbleIndex{db: db}, errors.Wrap(err, "setup db")
}

func (i *PebbleIndex) Close() error {
	return i.db.Close()
}

func (i *PebbleIndex) Flush() error {
	return i.db.Flush()
}

var writeOptions = pebble.NoSync

func (i *PebbleIndex) AddDir(dir *Dir, basePath string) error {
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
	batch := i.db.NewBatch()
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
		err = batch.Set(append(hashesPrefix, hash...), buf, nil)
		if err != nil {
			return errors.Wrap(err, "hashes.Put")
		}
		err = batch.Set(append(pathsPrefix, s.Path...), hash, nil)
		if err != nil {
			return errors.Wrap(err, "paths.Put")
		}
		putOps += 2
	}
	err = batch.Commit(writeOptions)
	if err != nil {
		return errors.Wrap(err, "batch.Commit")
	}
	return nil
}

func (i *PebbleIndex) LoadAfterHash(hash []byte, count int) ([]StoredSeal, error) {
	iterOptions := &pebble.IterOptions{
		LowerBound: hashesPrefix,
		UpperBound: keyUpperBound(hashesPrefix),
	}
	if len(hash) > 0 {
		iterOptions.LowerBound = keyUpperBound(append(hashesPrefix, hash...))
	}
	iter := i.db.NewIter(iterOptions)

	out := []StoredSeal{}
	for iter.First(); iter.Valid(); iter.Next() {
		// if len(hash) > 0 && bytes.Equal(iter.Key(), append(hashesPrefix, hash...)) {
		// 	continue
		// }
		err := iter.Error()
		if err != nil {
			return nil, errors.Wrap(err, "iter.Error")
		}
		// iter.Key()
		var s StoredSeal
		err = json.Unmarshal(iter.Value(), &s)
		if err != nil {
			return nil, errors.Wrap(err, "json.Unmarshal")
		}
		out = append(out, s)
		if len(out) >= count {
			break
		}
	}
	err := iter.Close()
	if err != nil {
		return nil, errors.Wrap(err, "iter.Close")
	}
	return out, nil
}

func keyUpperBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // no upper-bound
}
