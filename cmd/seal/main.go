package main

import (
	"fmt"
	"os"

	"github.com/mbertschler/seal"
)

func main() {
	err := seal.RootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
