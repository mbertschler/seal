package seal

import "bytes"

type Diff struct {
	Identical   bool
	HashChecked bool

	NameMatches      bool
	TotalSizeMatches bool
	ModifiedMatches  bool
	SHA256Matches    bool

	FilesMissing []*FileSeal
	FilesAdded   []*FileSeal

	Changes []*FileDiff
}

type FileDiff struct {
	Name string

	Want *FileSeal
	Have *FileSeal

	IsDirMatches    bool
	SizeMatches     bool
	ModifiedMatches bool
	SHA256Matches   bool
}

func DiffSeals(want, have *DirSeal, checkHash bool) *Diff {
	diff := &Diff{
		HashChecked:      checkHash,
		NameMatches:      want.Name == have.Name,
		TotalSizeMatches: want.TotalSize == have.TotalSize,
		ModifiedMatches:  want.Modified.Equal(have.Modified),
		SHA256Matches:    bytes.Equal(want.SHA256, have.SHA256),
	}
	if !checkHash {
		diff.SHA256Matches = true
	}

	type joined struct {
		want *FileSeal
		have *FileSeal
	}

	allFiles := map[string]joined{}
	for _, file := range want.Files {
		f := allFiles[file.Name]
		f.want = file
		allFiles[file.Name] = f
	}
	for _, file := range have.Files {
		f := allFiles[file.Name]
		f.have = file
		allFiles[file.Name] = f
	}

	for _, file := range allFiles {
		if file.want != nil && file.have == nil {
			diff.FilesMissing = append(diff.FilesMissing, file.want)
			continue
		}
		if file.have != nil && file.want == nil {
			diff.FilesAdded = append(diff.FilesAdded, file.have)
			continue
		}

		d := &FileDiff{
			IsDirMatches:    file.want.IsDir == file.have.IsDir,
			SizeMatches:     file.want.Size == file.have.Size,
			ModifiedMatches: file.want.Modified.Equal(file.have.Modified),
			SHA256Matches:   bytes.Equal(file.want.SHA256, file.have.SHA256),
		}
		if !checkHash {
			d.SHA256Matches = true
		}
		if d.IsDirMatches &&
			d.SizeMatches &&
			d.ModifiedMatches &&
			d.SHA256Matches {
			continue
		}

		d.Name = file.want.Name
		d.Want = file.want
		d.Have = file.have
		diff.Changes = append(diff.Changes, d)
	}

	if diff.NameMatches &&
		diff.TotalSizeMatches &&
		diff.ModifiedMatches &&
		diff.SHA256Matches &&
		len(diff.FilesAdded) == 0 &&
		len(diff.FilesMissing) == 0 &&
		len(diff.Changes) == 0 {
		diff.Identical = true
	}
	return diff
}
