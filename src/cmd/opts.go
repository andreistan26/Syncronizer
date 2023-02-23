package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

type ExchangeType int

const (
	LOCAL_EX ExchangeType = iota
	TCP_EX
)

type AddressPath struct {
	User     string
	Address  string
	Filepath string
}

type Options struct {
	exType ExchangeType
	Source AddressPath
	Dest   AddressPath

	Verbose  bool
	IsServer bool
}

func CreateMainCommand() (*cobra.Command, *Options) {
	options := &Options{}
	command := &cobra.Command{
		Use:   `sync [COMMAND]`,
		Short: `sync is a file syncronization tool`,
	}
	return command, options
}

func CreateSendCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   `send [OPTIONS] SRC DEST`,
		Short: `send files(SRC) to syncronize with a target(DEST)`,
		Args:  ArgsValidator(options),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Execute(options)
		},
	}

	command.Flags().BoolVarP(&options.Verbose, "verbose", "v", true, "increase verbosity")
	return command
}

func CreateServerCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   `server [OPTIONS]`,
		Short: `starts a server that listens for clients`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteStartServer(options)
		},
	}
	return command
}

func ArgsValidator(opts *Options) func(cmd *cobra.Command, args []string) error {
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
