package main

import (
	"fmt"
	"os"

	"github.com/mbertschler/seal"
)

func main() {
	err := seal.IndexBench()
	if err != nil {
		fmt.Println("indexbench:", err)
		os.Exit(1)
	}
}
