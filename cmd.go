package seal

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RootCmd is the what that should be executed by the seal command.
func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seal",
		Short: "Seal checks the integrity of your file archives",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, I'm seal! 🦭")
			cmd.Usage()
		},
	}
	cmd.AddCommand(sealCmd)
	cmd.AddCommand(verifyCmd)
	return cmd
}

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "seals all new files and directories",
	RunE:  runSealCmd,
}

func runSealCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("need at least one path argument to seal")
	}

	for _, arg := range args {
		PrintSealing = true
		_, err := SealPath(arg)
		if err != nil {
			return errors.Wrap(err, "SealPath")
		}
	}
	return nil
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "verifies all files and directories against seal files",
	RunE:  runVerifyCmd,
}

func runVerifyCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("need at least one path argument to verify")
	}

	for _, arg := range args {
		PrintVerify = true
		err := VerifyPath(arg)
		if err != nil {
			return errors.Wrap(err, "VerifyPath")
		}
	}
	return nil
}
