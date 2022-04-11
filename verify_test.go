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

	dirs, err := SealPath(TestDir)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)

	dirs, err = VerifyPath(TestDir, false)
	require.NoError(t, err)

	for _, dir := range dirs {
		assert.True(t, dir.quick.Identical)
		assert.True(t, dir.hash.Identical)
	}

	randomFile(t, TestDir+"/a.txt", 4) // different content
	randomFile(t, TestDir+"/b.txt", 5) // new file
	assert.NoError(t, os.Remove(TestDir+"/sub/d.txt"))

	dirs, err = VerifyPath(TestDir, false)
	require.NoError(t, err)

	assert.Equal(t, 0, len(dirs[0].quick.FilesAdded))
	assert.Equal(t, 0, len(dirs[0].hash.FilesAdded))
	assert.Equal(t, 1, len(dirs[0].quick.FilesMissing))
	assert.Equal(t, 1, len(dirs[0].hash.FilesMissing))
	assert.Equal(t, 0, len(dirs[0].quick.FilesChanged))
	assert.Equal(t, 0, len(dirs[0].hash.FilesChanged))

	assert.Equal(t, "d.txt", dirs[0].quick.FilesMissing[0].Name)
	assert.Equal(t, "d.txt", dirs[0].hash.FilesMissing[0].Name)

	assert.Equal(t, 1, len(dirs[1].quick.FilesAdded))
	assert.Equal(t, 1, len(dirs[1].hash.FilesAdded))
	assert.Equal(t, 0, len(dirs[1].quick.FilesMissing))
	assert.Equal(t, 0, len(dirs[1].hash.FilesMissing))
	assert.Equal(t, 1, len(dirs[1].quick.FilesChanged))
	assert.Equal(t, 1, len(dirs[1].hash.FilesChanged))

	assert.Equal(t, "b.txt", dirs[1].quick.FilesAdded[0].Name)
	assert.Equal(t, "b.txt", dirs[1].hash.FilesAdded[0].Name)
	assert.Equal(t, "a.txt", dirs[1].quick.FilesChanged[0].Have.Name)
	assert.Equal(t, false, dirs[1].quick.FilesChanged[0].ModifiedMatches)
	assert.Equal(t, "a.txt", dirs[1].hash.FilesChanged[0].Have.Name)
	assert.Equal(t, false, dirs[1].hash.FilesChanged[0].SHA256Matches)
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
