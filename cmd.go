package seal

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	IndexFile     string

	WriteLock sync.Mutex
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
					Before, err = time.ParseInLocation(layout, beforeFlag, time.Local)
					if err == nil {
						log.Println("filtering out directories sealed after", Before)
						break
					}
				}
			}

			go func() {
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
				s := <-sigs
				log.Printf("got %v signal, exiting", s)
				WriteLock.Lock()
				os.Exit(0)
			}()
			return err
		},
	}
	cmd.AddCommand(sealCmd)
	cmd.AddCommand(verifyCmd)
	cmd.AddCommand(indexCmd)

	cmd.PersistentFlags().StringVarP(&beforeFlag, "before", "b", "", "ignore directories sealed after this time")
	cmd.PersistentFlags().DurationVarP(&PrintInterval, "interval", "i", time.Minute, "interval at which progress is reported")
	cmd.PersistentFlags().StringVarP(&IndexFile, "file", "f", "", "index file path")
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
	for _, path := range args {
		PrintSealing = true
		PrintIndexProgress = true
		_, err := SealPath(path)
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
	for _, path := range args {
		PrintVerify = true
		PrintIndexProgress = true
		printDifferences := true
		_, err := VerifyPath(path, printDifferences)
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
	start := time.Now()
	for _, path := range args {
		err := IndexPath(path, IndexFile)
		if err != nil {
			return errors.Wrap(err, "IndexPath")
		}
	}
	log.Println("ran for", time.Since(start))
	return nil
}
