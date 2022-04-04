package seal

import (
	"log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RootCmd is the what that should be executed by the seal command.
var RootCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal checks the integrity of your file archives",
	RunE:  runRootCmd,
}

func runRootCmd(cmd *cobra.Command, args []string) error {
	log.Println("Hello, I'm seal! 🦭")

	if len(args) == 0 {
		return errors.New("need at least one path argument to seal")
	}

	for _, arg := range args {
		err := SealPath(arg)
		if err != nil {
			return errors.Wrap(err, "sealDir")
		}
	}
	return nil
}
