package seal

import "bytes"

// Diff holds the differences between two DirSeals.
type Diff struct {
	Identical   bool
	HashChecked bool

	NameMatches      bool
	TotalSizeMatches bool
	ModifiedMatches  bool
	SHA256Matches    bool

	FilesMissing []*FileSeal
	FilesAdded   []*FileSeal
	FilesChanged []*FileDiff
}

// FileDiff holds the differences between two FileSeals.
type FileDiff struct {
	Name string

	Want *FileSeal
	Have *FileSeal

	IsDirMatches    bool
	SizeMatches     bool
	ModifiedMatches bool
	SHA256Matches   bool
}

// DiffSeals finds all differences between two DirSeals.
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

	diffFiles(diff, want, have, checkHash)

	if diff.NameMatches &&
		diff.TotalSizeMatches &&
		diff.ModifiedMatches &&
		diff.SHA256Matches &&
		len(diff.FilesAdded) == 0 &&
		len(diff.FilesMissing) == 0 &&
		len(diff.FilesChanged) == 0 {
		diff.Identical = true
	}
	return diff
}

// diffFiles adds the differences between the file slices of
// the two DirSeals to the diff.
func diffFiles(d *Diff, want, have *DirSeal, checkHash bool) {
	type joinedSeals struct {
		want *FileSeal
		have *FileSeal
	}

	// join files from both seals in one map for easy handling
	allFiles := map[string]joinedSeals{}
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
			d.FilesMissing = append(d.FilesMissing, file.want)
			continue
		}
		if file.have != nil && file.want == nil {
			d.FilesAdded = append(d.FilesAdded, file.have)
			continue
		}

		fd := &FileDiff{
			IsDirMatches:    file.want.IsDir == file.have.IsDir,
			SizeMatches:     file.want.Size == file.have.Size,
			ModifiedMatches: file.want.Modified.Equal(file.have.Modified),
			SHA256Matches:   bytes.Equal(file.want.SHA256, file.have.SHA256),
		}
		if !checkHash {
			fd.SHA256Matches = true
		}
		if fd.IsDirMatches &&
			fd.SizeMatches &&
			fd.ModifiedMatches &&
			fd.SHA256Matches {
			continue
		}

		fd.Name = file.want.Name
		fd.Want = file.want
		fd.Have = file.have
		d.FilesChanged = append(d.FilesChanged, fd)
	}
}
