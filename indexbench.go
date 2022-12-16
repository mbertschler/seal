package seal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

const (
	filesPerDir = 10
	dirsPerDir  = 5
)

func IndexBench() error {
	dirs := generateDirs(30)
	buf, err := json.Marshal(dirs)
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateDirs(numFiles int) []Dir {
	basedir := Dir{Path: "basedir",
		Seal: &DirSeal{
			Name:      "basedir",
			TotalSize: int64(rand.Intn(1e7)),
			SHA256:    []byte(randomString(32)),
			Modified:  time.Now(),
			Sealed:    time.Now(),
		},
	}

	generator := dirGenerator{
		output:  []Dir{basedir},
		current: &basedir,
		toFill:  []*Dir{&basedir},
	}

	for i := 0; i < numFiles; i++ {
		generator.generateFile()
	}

	return generator.output
}

type dirGenerator struct {
	output  []Dir
	current *Dir
	toFill  []*Dir
}

func (g *dirGenerator) generateFile() {
	if len(g.current.Seal.Files) >= filesPerDir {
		g.nextDir()
	}
	g.current.Seal.Files = append(g.current.Seal.Files, &FileSeal{
		Name:     randomString(10),
		Size:     int64(rand.Intn(1e6)),
		SHA256:   []byte(randomString(32)),
		Modified: time.Now(),
		Sealed:   time.Now(),
	})
}

func (g *dirGenerator) nextDir() {
	// fmt.Println("nextDir:", g.current.Path, g.toFill)
	// defer fmt.Println("deferred:", g.current.Path)
	g.current = g.toFill[0]
	if len(g.current.Seal.Files) >= filesPerDir+dirsPerDir {
		g.toFill = g.toFill[1:]
		g.current = g.toFill[0]
	}
	name := randomString(10)
	g.output = append(g.output, Dir{
		Path:  g.current.Path + "/" + name,
		Depth: g.current.Depth + 1,
		Seal: &DirSeal{
			Name:      name,
			TotalSize: int64(rand.Intn(1e7)),
			SHA256:    []byte(randomString(32)),
			Modified:  time.Now(),
			Sealed:    time.Now(),
		},
	})
	g.current.Seal.Files = append(g.current.Seal.Files, &FileSeal{
		Name:     name,
		IsDir:    true,
		Size:     int64(rand.Intn(1e6)),
		SHA256:   []byte(randomString(32)),
		Modified: time.Now(),
		Sealed:   time.Now(),
	})

	g.current = &g.output[len(g.output)-1]
	g.toFill = append(g.toFill, g.current)
}
