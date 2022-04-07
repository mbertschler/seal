package seal

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/fatih/color"
)

// Diff holds the differences between two DirSeals.
type Diff struct {
	Identical   bool
	HashChecked bool

	Want *DirSeal
	Have *DirSeal

	NameMatches      bool
	TotalSizeMatches bool
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
		SHA256Matches:    bytes.Equal(want.SHA256, have.SHA256),
	}
	if !checkHash {
		diff.SHA256Matches = true
	}

	diffFiles(diff, want, have, checkHash)

	if diff.NameMatches &&
		diff.TotalSizeMatches &&
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
		if !file.exists() {
			continue
		}
		f := allFiles[file.Name]
		f.want = file
		allFiles[file.Name] = f
	}
	for _, file := range have.Files {
		if !file.exists() {
			continue
		}
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

		if checkHash {
			fd.ModifiedMatches = true
		} else {
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
	if !d.SHA256Matches {
		if len(meta) != 0 {
			meta += ", "
		}
		meta += "SHA256 doesn't match"
	}
	if len(meta) > 0 {
		log.Println(color.RedString("dir differs: %q %s", d.Have.Name, meta))
	}

	for _, f := range d.FilesAdded {
		log.Println(color.GreenString("added file: %q", f.Name))
	}
	for _, f := range d.FilesMissing {
		log.Println(color.RedString("missing file: %q", f.Name))
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
			if len(meta) != 0 {
				meta += ", "
			}
			differences += fmt.Sprintf("Modified is:%s want:%s",
				d.Have.Modified.Format(time.RFC3339), d.Want.Modified.Format(time.RFC3339))
		}
		if !f.SHA256Matches {
			if len(differences) != 0 {
				differences += ", "
			}
			differences += "SHA256 doesn't match"
		}
		log.Println(color.RedString("file differs: %q %s", f.Want.Name, differences))
	}
}
