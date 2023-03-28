package seal

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	_ "github.com/mattn/go-sqlite3"
)

const StorageTypeSQLite StorageType = "sqlite"

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
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS seal_path ON seals(path)")
	if err != nil {
		return nil, errors.Wrap(err, "create path index")
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

	tx, err := i.db.Begin()
	if err != nil {
		return errors.Wrap(err, "db.BeginTx")
	}
	defer tx.Rollback()

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

		const insert = `INSERT INTO seals (hash, path, json) VALUES ($1, $2, $3)
		ON CONFLICT (hash) DO UPDATE SET path = $2, json = $3;`
		_, err = tx.Exec(insert, hashString, s.Path, buf)
		if err != nil {
			return errors.Wrap(err, "insert")
		}
		putOps++
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "tx.Commit")
	}

	return nil
}

func (i *SqliteIndex) LoadAfterHash(hash []byte, count int) ([]StoredSeal, error) {
	hashString := base64.RawStdEncoding.EncodeToString(hash)
	rows, err := i.db.Query(`SELECT json FROM seals
	WHERE hash > $1 ORDER BY hash ASC LIMIT $2;`, hashString, count)
	if err != nil {
		return nil, errors.Wrap(err, "db.Query")
	}
	defer rows.Close()

	out := []StoredSeal{}
	for rows.Next() {
		var buf []byte
		err = rows.Scan(&buf)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		var s StoredSeal
		err = json.Unmarshal(buf, &s)
		if err != nil {
			return nil, errors.Wrap(err, "json.Unmarshal")
		}
		out = append(out, s)
	}
	return out, nil
}
