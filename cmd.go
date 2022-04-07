package seal

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	beforeFlag  string
	timeLayouts = []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	Before        time.Time
	PrintInterval time.Duration
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if beforeFlag != "" {
				for _, layout := range timeLayouts {
					Before, err = time.Parse(layout, beforeFlag)
					if err == nil {
						log.Println("filtering out directories sealed after", Before)
						break
					}
				}
			}
			return err
		},
	}
	cmd.AddCommand(sealCmd)
	cmd.AddCommand(verifyCmd)
	cmd.AddCommand(indexCmd)

	cmd.PersistentFlags().StringVarP(&beforeFlag, "before", "b", "", "ignore directories sealed after this time")
	cmd.PersistentFlags().DurationVarP(&PrintInterval, "interval", "i", time.Minute, "interval at which progress is reported")
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

	start := time.Now()
	for _, arg := range args {
		PrintSealing = true
		PrintIndexProgress = true
		_, err := SealPath(arg)
		if err != nil {
			return errors.Wrap(err, "SealPath")
		}

	}
	log.Println("ran for", time.Since(start))
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

	start := time.Now()
	for _, arg := range args {
		PrintVerify = true
		PrintIndexProgress = true
		printDifferences := true
		_, err := VerifyPath(arg, printDifferences)
		if err != nil {
			return errors.Wrap(err, "VerifyPath")
		}
	}
	log.Println("ran for", time.Since(start))
	return nil
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "indexes all directories and prints stats",
	RunE:  runIndexCmd,
}

func runIndexCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("need at least one path argument to index")
	}

	PrintIndexProgress = true
	for _, path := range args {
		log.Println("indexing", path)
		start := time.Now()
		dirs, err := indexDirectories(path)
		if err != nil {
			return errors.Wrap(err, "indexDirectories")
		}
		log.Println("loaded", len(dirs), "directories in", time.Since(start))
	}
	return nil
}
