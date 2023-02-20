package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

type Options struct {
	Source string
	Dest   string

	Verbose bool
}

func CreateMainCommand() *cobra.Command {
	options := &Options{}

	command := &cobra.Command{
		Use:   `sync SRC DEST`,
		Short: `sync is a file syncronization tool`,
		Args:  ArgsValidator(options),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Execute(*options)
			// return nil
		},
	}

	command.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", true, "increase verbosity")

	return command
}

func ArgsValidator(opts *Options) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		println(args[0])
		println(args[1])
		println(os.Getwd())
		if len(args) < 2 {
			return errors.New("you need to provide a source and a destination")
		}
		if _, err = os.Stat(args[1]); err != nil {
			return errors.New("source file does not exist")
		}
		opts.Source = args[0]
		opts.Dest = args[1]
		return nil
	}
}
