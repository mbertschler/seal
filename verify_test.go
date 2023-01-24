package seal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	expected := SetupTestDir(t)

	dirs, err := SealPath(TestDir, nil)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)

	dirs, err = VerifyPath(TestDir, false, nil)
	require.NoError(t, err)

	for _, dir := range dirs {
		assert.True(t, dir.QuickDiff.Identical)
		assert.True(t, dir.HashDiff.Identical)
	}

	randomFile(t, TestDir+"/a.txt", 4) // different content
	randomFile(t, TestDir+"/b.txt", 5) // new file
	assert.NoError(t, os.Remove(TestDir+"/sub/d.txt"))

	dirs, err = VerifyPath(TestDir, false, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, len(dirs[0].QuickDiff.FilesAdded))
	assert.Equal(t, 0, len(dirs[0].HashDiff.FilesAdded))
	assert.Equal(t, 1, len(dirs[0].QuickDiff.FilesMissing))
	assert.Equal(t, 1, len(dirs[0].HashDiff.FilesMissing))
	assert.Equal(t, 0, len(dirs[0].QuickDiff.FilesChanged))
	assert.Equal(t, 0, len(dirs[0].HashDiff.FilesChanged))

	assert.Equal(t, "d.txt", dirs[0].QuickDiff.FilesMissing[0].Name)
	assert.Equal(t, "d.txt", dirs[0].HashDiff.FilesMissing[0].Name)

	assert.Equal(t, 1, len(dirs[1].QuickDiff.FilesAdded))
	assert.Equal(t, 1, len(dirs[1].HashDiff.FilesAdded))
	assert.Equal(t, 0, len(dirs[1].QuickDiff.FilesMissing))
	assert.Equal(t, 0, len(dirs[1].HashDiff.FilesMissing))
	assert.Equal(t, 1, len(dirs[1].QuickDiff.FilesChanged))
	assert.Equal(t, 1, len(dirs[1].HashDiff.FilesChanged))

	assert.Equal(t, "b.txt", dirs[1].QuickDiff.FilesAdded[0].Name)
	assert.Equal(t, "b.txt", dirs[1].HashDiff.FilesAdded[0].Name)
	assert.Equal(t, "a.txt", dirs[1].QuickDiff.FilesChanged[0].Have.Name)
	assert.Equal(t, false, dirs[1].QuickDiff.FilesChanged[0].ModifiedMatches)
	assert.Equal(t, "a.txt", dirs[1].HashDiff.FilesChanged[0].Have.Name)
	assert.Equal(t, false, dirs[1].HashDiff.FilesChanged[0].SHA256Matches)
}

func TestBasepath(t *testing.T) {
	fullPath := "/Volumes/Membox/photos/P1070520.RW2"

	basePath := "/Volumes/Membox"
	out, err := filepath.Rel(basePath, fullPath)
	assert.NoError(t, err)
	assert.Equal(t, "photos/P1070520.RW2", out)

	basePath = "/Volumes/Membox/"
	out, err = filepath.Rel(basePath, fullPath)
	assert.NoError(t, err)
	assert.Equal(t, "photos/P1070520.RW2", out)

	fullPath = "/Volumes/Membox/photos/"
	out, err = filepath.Rel(basePath, fullPath)
	assert.NoError(t, err)
	assert.Equal(t, "photos", out)

	fullPath = "/Volumes/Membox/photos"
	out, err = filepath.Rel(basePath, fullPath)
	assert.NoError(t, err)
	assert.Equal(t, "photos", out)
}
