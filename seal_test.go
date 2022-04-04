package seal

import (
	"encoding/base64"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func SetupTestDir(t testing.TB) {
	assert.NoError(t, os.RemoveAll("./testdir"))
	assert.NoError(t, os.MkdirAll("./testdir/ground/upper", 0755))
	randomFile(t, "./testdir/base.txt", 1)
	randomFile(t, "./testdir/exponent.txt", 2)
	randomFile(t, "./testdir/ground/subfolder.txt", 3)
	randomFile(t, "./testdir/ground/duplicate.txt", 4)
	randomFile(t, "./testdir/ground/upper/deep.txt", 4)
}

func randomFile(t testing.TB, path string, seed int64) {
	src := rand.New(rand.NewSource(seed))
	buf := make([]byte, 1992)
	src.Read(buf)
	text := base64.StdEncoding.EncodeToString(buf)
	assert.NoError(t, ioutil.WriteFile(path, []byte(text), 0644))
}

func TestSetup(t *testing.T) {
	SetupTestDir(t)
}
