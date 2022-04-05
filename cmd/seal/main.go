package main

import (
	"os"

	"github.com/mbertschler/seal"
)

func main() {
	err := seal.RootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}
