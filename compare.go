package seal

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func compareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "compare indexes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("need two index paths to compare indices")
			}
			return CompareIndices(args[0], args[1])
		},
	}
	return cmd
}

func CompareIndices(pathA, pathB string) error {
	PrintIndexProgress = true

	start := time.Now()
	indexA, err := LoadIndex(pathA, StorageTypeSQLite)
	if err != nil {
		return errors.Wrap(err, "LoadIndex pathA")
	}

	log.Println("len by path", len(indexA.ByPath), "by hash", len(indexA.ByHash), "dirs", len(indexA.Dirs))

	// count := 0
	// for key, seal := range indexA.ByPath {
	// 	log.Printf("key %q path %q\n", key, seal.Path)
	// 	printStoredSeal(seal)
	// 	count++
	// 	if count > 10 {
	// 		break
	// 	}
	// }

	indexB, err := LoadIndex(pathB, StorageTypeSQLite)
	if err != nil {
		return errors.Wrap(err, "LoadIndex pathB")
	}
	log.Println("loaded both indices after", time.Since(start))

	rootA := indexA.ByPath["."]
	rootB := indexB.ByPath["."]
	// printStoredSeal(rootA)
	// printStoredSeal(rootB)
	// fmt.Println()

	diff := DiffSeals(rootA.Dir, rootB.Dir, true)
	log.Println("Differences:")
	diff.PrintDifferences()

	compareDirsWithIndex(indexA.Dirs, indexB)
	compareDirsWithIndex(indexB.Dirs, indexA)
	log.Println("done comparing indices after", time.Since(start))
	return nil
}

func printStoredSeal(seal *StoredSeal) {
	fmt.Printf("seal for %q is a ", seal.Path)
	if seal.Dir != nil {
		fmt.Println("dir")
		printDirSeal(seal.Dir)
	} else {
		fmt.Println("file")
		printFileSeal(seal.File)
	}
}

func printFileSeal(seal *FileSeal) {
	fmt.Printf("%#v\n", seal)
}

func printDirSeal(seal *DirSeal) {
	fmt.Printf("%#v\n", seal)
	for _, file := range seal.Files {
		fmt.Printf("    %#v\n", file)
	}
}

func compareDirsWithIndex(dirs []Dir, index *LoadedIndex) error {
	return nil
}
