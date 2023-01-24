package seal

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	filesPerDir = 10
	dirsPerDir  = 5
)

func indexBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "indexbench",
		Short: "indexing benchmarks",
	}

	writeCmd := &cobra.Command{
		Use:   "write",
		Short: "index write benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			return IndexBenchWrite()
		},
	}
	cmd.AddCommand(writeCmd)

	readCmd := &cobra.Command{
		Use:   "read",
		Short: "index read benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("need a path argument to read index")
			}
			return IndexBenchRead(args[0])
		},
	}
	cmd.AddCommand(readCmd)

	return cmd
}

func IndexBenchWrite() error {
	start := time.Now()
	dirs := generateDirs(10e3)
	log.Println("generated", len(dirs), "directories with seals in", time.Since(start))

	start = time.Now()
	indexFile := "./benchindex_bolt.out"
	err := os.Remove(indexFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	path := "basedir"
	err = DirsToIndex(indexFile, dirs, path, IndexBoltDB)
	took := time.Since(start)
	log.Println("indexed", len(dirs), "directories with seals in", took, "with", putOps, "writes")
	log.Printf("BoltDB %v average write time", time.Duration(float64(took)/float64(putOps)))
	if err != nil {
		return err
	}

	putOps = 0

	start = time.Now()
	indexFile = "./benchindex_sqlite.out"
	err = os.Remove(indexFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = DirsToIndex(indexFile, dirs, path, IndexSQLite)
	took = time.Since(start)
	log.Println("indexed", len(dirs), "directories with seals in", took, "with", putOps, "writes")
	log.Printf("SQLite %v average write time", time.Duration(float64(took)/float64(putOps)))
	if err != nil {
		return err
	}

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

func IndexBenchRead(path string) error {
	log.Println("loading", path, "index")
	PrintIndexProgress = true
	start := time.Now()
	index, err := LoadIndex(path, IndexSQLite)
	if err != nil {
		return errors.Wrap(err, "LoadIndex")
	}
	log.Println("loading", len(index.ByHash), "hashes took", time.Since(start))
	return nil
}
