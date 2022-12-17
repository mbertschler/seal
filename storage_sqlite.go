package seal

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	_ "github.com/mattn/go-sqlite3"
)

const IndexSQLite StorageType = "sqlite"

type SqliteIndex struct {
	db *sql.DB
}

func OpenSqlite(indexPath string) (*SqliteIndex, error) {
	db, err := sql.Open("sqlite3", indexPath)
	if err != nil {
		return nil, errors.Wrap(err, "sql.Open")
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS seals (hash TEXT PRIMARY KEY, path TEXT, json BLOB);")
	if err != nil {
		return nil, errors.Wrap(err, "create table")
	}
	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS seal_path ON seals(path)")
	if err != nil {
		return nil, errors.Wrap(err, "create index")
	}

	return &SqliteIndex{db: db}, errors.Wrap(err, "setup db")
}

func (i *SqliteIndex) Close() error {
	return i.db.Close()
}

func (i *SqliteIndex) AddDir(dir *Dir, basePath string) error {
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

		hashString := base64.RawStdEncoding.EncodeToString(hash)

		const insert = `INSERT INTO seals (hash, path, json) VALUES ($1, $2, $3);`
		_, err = i.db.Exec(insert, hashString, s.Path, buf)
		if err != nil {
			return errors.Wrap(err, "insert")
		}
		putOps++
	}

	return nil
}
