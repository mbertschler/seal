package seal

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RootCmd is the what that should be executed by the seal command.
var RootCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal checks the integrity of your file archives",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("I'm seal 🦭")
	},
}
