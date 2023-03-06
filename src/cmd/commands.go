package cmd

import (
	"errors"
	"os"

	"github.com/andreistan26/sync/src/options"
	"github.com/spf13/cobra"
)

func CreateMainCommand() (*cobra.Command, *options.Options) {
	opts := &options.Options{}
	command := &cobra.Command{
		Use:   `sync [COMMAND]`,
		Short: `sync is a file syncronization tool`,
	}
	return command, opts
}

func CreateSendCommand(opts *options.Options) *cobra.Command {
	command := &cobra.Command{
		Use:   `send [opts] SRC DEST`,
		Short: `send files(SRC) to syncronize with a target(DEST)`,
		Args:  ArgsValidator(opts),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Execute(opts)
		},
	}

	command.Flags().BoolVarP(&opts.Verbose, "verbose", "v", true, "increase verbosity")
	command.Flags().IntVar(&opts.Port, "Port", options.DEFAULT_PORT, "specify address port")
	return command
}

func CreateServerCommand() *cobra.Command {
	opts := &options.ServerOptions{}
	command := &cobra.Command{
		Use:   `server [OPTIONS]`,
		Short: `starts a server that listens for clients`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteStartServer(opts)
		},
	}

	command.Flags().IntVar(&opts.Port, "Port", options.DEFAULT_PORT, "specify port")
	return command
}

func ArgsValidator(opts *options.Options) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 2 {
			return errors.New("you need to provide a source and a destination")
		}
		if _, err = os.Stat(args[0]); err != nil {
			return errors.New("source file does not exist")
		}
		opts.ParseArgument(args)
		return nil
	}
}
