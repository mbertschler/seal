package seal

import (
	"bytes"
	"fmt"
	"log"
)

// Diff holds the differences between two DirSeals.
type Diff struct {
	Identical   bool
	HashChecked bool

	Want *DirSeal
	Have *DirSeal

	NameMatches      bool
	TotalSizeMatches bool
	ModifiedMatches  bool
	SHA256Matches    bool

	FilesAdded   []*FileSeal
	FilesMissing []*FileSeal
	FilesChanged []*FileDiff
}

// FileDiff holds the differences between two FileSeals.
type FileDiff struct {
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
		HashChecked: checkHash,
		Want:        want,
		Have:        have,

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

		fd.Want = file.want
		fd.Have = file.have
		d.FilesChanged = append(d.FilesChanged, fd)
	}
}

// PrintDifferences prints the differences between two seals.
func (d *Diff) PrintDifferences() {
	if d.Identical {
		return
	}
	var meta string
	if !d.NameMatches {
		meta += fmt.Sprintf("Name is:%q want:%q", d.Have.Name, d.Want.Name)
	}
	if !d.TotalSizeMatches {
		if len(meta) != 0 {
			meta += ", "
		}
		meta += fmt.Sprintf("TotalSize is:%d want:%d", d.Have.TotalSize, d.Want.TotalSize)
	}
	if !d.ModifiedMatches {
		if len(meta) != 0 {
			meta += ", "
		}
		meta += fmt.Sprintf("Modified is:%v want:%v", d.Have.Modified, d.Want.Modified)
	}
	if !d.SHA256Matches {
		if len(meta) != 0 {
			meta += ", "
		}
		meta += fmt.Sprintf("SHA256 is:%x want:%x", d.Have.SHA256, d.Want.SHA256)
	}
	if len(meta) > 0 {
		log.Println("dir differs:", meta)
	}

	for _, f := range d.FilesAdded {
		log.Printf("added file: %q", f.Name)
	}
	for _, f := range d.FilesMissing {
		log.Printf("missing file: %q", f.Name)
	}
	for _, f := range d.FilesChanged {
		var differences string
		if !f.IsDirMatches {
			differences += fmt.Sprintf("IsDir is:%t want:%t", f.Have.IsDir, f.Want.IsDir)
		}
		if !f.SizeMatches {
			if len(differences) != 0 {
				differences += ", "
			}
			differences += fmt.Sprintf("Size is:%d want:%d", f.Have.Size, f.Want.Size)
		}
		if !f.ModifiedMatches {
			if len(differences) != 0 {
				differences += ", "
			}
			differences += fmt.Sprintf("Modified is:%v want:%v", f.Have.Modified, f.Want.Modified)
		}
		if !f.SHA256Matches {
			if len(differences) != 0 {
				differences += ", "
			}
			differences += fmt.Sprintf("SHA256 is:%x want:%x", f.Have.SHA256, f.Want.SHA256)
		}
		log.Printf("file differs: %q %s", f.Want.Name, differences)
	}
}
