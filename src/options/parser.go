package options

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidAddress = errors.New("invalid address or file path from argument")
)

func (addrPath *AddressPath) parse(arg string) error {
	switch res := strings.Split(arg, ":"); len(res) {
	// also specified address
	case 2:
		addr := strings.Split(res[0], "@")
		if len(addr) != 2 {
			fmt.Print(ErrInvalidAddress)
			return ErrInvalidAddress
		}
		addrPath.Filepath = res[1]
		addrPath.User = addr[0]
		addrPath.Address = addr[1]
	case 1:
		addrPath.Filepath = res[0]
	default:
		fmt.Print(ErrInvalidAddress)
		return ErrInvalidAddress
	}
	return nil
}

func (addrPath *AddressPath) ParseSource(arg string) error {
	err := addrPath.parse(arg)
	if addrPath.User != "" || addrPath.Address != "" {
		return ErrInvalidAddress
	}
	return err
}

func (addrPath *AddressPath) ParseDest(arg string) (ExchangeType, error) {
	err := addrPath.parse(arg)
	if err == nil && addrPath.Address != "" {
		return TCP_EX, nil
	}
	return LOCAL_EX, nil
}

// assumes that the lenght is at least 2
func (opts *Options) ParseArgument(arg []string) {
	err := opts.Source.ParseSource(arg[0])
	if err != nil {
		panic(err)
	}

	opts.ExType, err = opts.Dest.ParseDest(arg[1])
	if err != nil {
		panic(err)
	}
}
