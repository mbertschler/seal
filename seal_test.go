package seal

import (
	"encoding/base64"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TestDir = "./testdir"

type ExpectedDir struct {
	Name      string
	TotalSize int64
	SHA256    string
	Files     []*ExpectedFile
}

func (e *ExpectedDir) ResetFound() {
	for _, f := range e.Files {
		f.Found = 0
	}
}

func (e *ExpectedDir) AllFoundOnce(t *testing.T) {
	for _, f := range e.Files {
		assert.Equal(t, 1, f.Found, "%+v", f)
	}
}

type ExpectedFile struct {
	Name       string
	IsDir      bool
	Size       int64
	Deleted    bool
	OldVersion bool
	SHA256     string

	Found int
}

func SetupTestDir(t testing.TB) map[string]ExpectedDir {
	assert.NoError(t, os.RemoveAll(TestDir))
	assert.NoError(t, os.MkdirAll(TestDir+"/sub", 0755))

	randomFile(t, TestDir+"/a.txt", 1)
	randomFile(t, TestDir+"/sub/c.txt", 2) // duplicate to b.txt
	randomFile(t, TestDir+"/sub/d.txt", 3)

	testdir := ExpectedDir{
		Name:      "testdir",
		TotalSize: 7968,
		SHA256:    "naZ2L2KucwkQkMFqhVntskAEkWdENOVdZbParSGnRg0=",
		Files: []*ExpectedFile{
			{
				Name:   "a.txt",
				Size:   2656,
				SHA256: "Y5rUr7x9uXN+Bx8H6Lr/9U2ft9En6/0g0t4GS/TvR3c=",
			},
			{
				Name:   "sub",
				IsDir:  true,
				Size:   5312,
				SHA256: "/f/rUwGRu1LxK8p5ug/rTigixEdUBltK/TqLjhdQLXE=",
			},
		},
	}

	sub := ExpectedDir{
		Name:      "sub",
		TotalSize: 5312,
		SHA256:    "/f/rUwGRu1LxK8p5ug/rTigixEdUBltK/TqLjhdQLXE=",
		Files: []*ExpectedFile{
			{
				Name:   "c.txt",
				Size:   2656,
				SHA256: "3BFVDqrHcM6JnYcYc9qq6Cw1VTDNITwq6cqWrfJKffU=",
			},
			{
				Name:   "d.txt",
				Size:   2656,
				SHA256: "C9olDOhodiVkC6otxyito/wnOQxoFfRS9iNj2Pr7uJc=",
			},
		},
	}

	return map[string]ExpectedDir{
		"testdir":     testdir,
		"testdir/sub": sub,
	}
}

func randomFile(t testing.TB, path string, seed int64) {
	src := rand.New(rand.NewSource(seed))
	buf := make([]byte, 1992)
	src.Read(buf)
	text := base64.StdEncoding.EncodeToString(buf)
	assert.NoError(t, ioutil.WriteFile(path, []byte(text), 0644))
}

func TestSeal(t *testing.T) {
	expected := SetupTestDir(t)

	dirs, err := SealPath(TestDir)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)

	dirs, err = SealPath(TestDir)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)

	randomFile(t, TestDir+"/a.txt", 4) // different content
	randomFile(t, TestDir+"/b.txt", 5) // new file
	assert.NoError(t, os.Remove(TestDir+"/sub/d.txt"))

	d := expected["testdir"]
	d.TotalSize = 7968
	d.SHA256 = "dGVFwhVfMYXrTUsgcK3PB4lyzHRj+pdz3yi1joJOQJE="
	d.Files = append(d.Files, &ExpectedFile{
		Name:   "a.txt",
		Size:   2656,
		SHA256: "YZAevXtzTdDGGNvX0MbTLNzFluCE9qGDxrNPcwqk00s=",
	})
	d.Files = append(d.Files, &ExpectedFile{
		Name:   "b.txt",
		Size:   2656,
		SHA256: "z+uofBw894tArWoaetwMUe3DWZDVBMujCgNDaH5LVUY=",
	})
	d.Files = append(d.Files, &ExpectedFile{
		Name:   "sub",
		IsDir:  true,
		Size:   2656,
		SHA256: "SyHqJ6w1qEa579EEF+TYE45WZsbSG39K/l6dbpYaFw8=",
	})
	d.Files[0].OldVersion = true
	d.Files[1].OldVersion = true
	expected["testdir"] = d

	d = expected["testdir/sub"]
	d.TotalSize = 2656
	d.SHA256 = "SyHqJ6w1qEa579EEF+TYE45WZsbSG39K/l6dbpYaFw8="
	d.Files[1].Deleted = true
	expected["testdir/sub"] = d

	dirs, err = SealPath(TestDir)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)

	dirs, err = SealPath(TestDir)
	require.NoError(t, err)
	checkDirs(t, dirs, expected)
}

func checkDirs(t *testing.T, dirs []*dir, expected map[string]ExpectedDir) {
	for _, dir := range dirs {
		e, ok := expected[dir.path]
		require.True(t, ok)
		assert.Equal(t, e.Name, dir.seal.Name)
		assert.Equal(t, e.TotalSize, dir.seal.TotalSize)
		assert.Equal(t, e.SHA256, Base64(dir.seal.SHA256))
		for _, file := range dir.seal.Files {
			findFile(t, file, e.Files)
		}
		assert.Equal(t, len(e.Files), len(dir.seal.Files))
		e.AllFoundOnce(t)
		e.ResetFound()
	}
}

func findFile(t *testing.T, file *FileSeal, expected []*ExpectedFile) {
	for _, f := range expected {
		if file.Name == f.Name &&
			file.IsDir == f.IsDir &&
			file.Size == f.Size &&
			file.Deleted == f.Deleted &&
			file.OldVersion == f.OldVersion &&
			Base64(file.SHA256) == f.SHA256 {
			f.Found++
		}
	}
}
