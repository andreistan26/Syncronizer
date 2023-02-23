package cmd

import (
	"github.com/andreistan26/sync/src/file_level"
	transport "github.com/andreistan26/sync/src/transfer_level"
)

func Execute(options *Options) error {
	switch options.exType {
	case LOCAL_EX:
		println("herea")
		return ExecuteHostExchange(options)
	case TCP_EX:
		println("erea")
		return ExecuteTCPExchange(options)
	default:
		return nil
	}
}

func ExecuteHostExchange(options *Options) error {
	sf := file_level.CreateSourceFile(options.Source.Filepath)
	rf := file_level.CreateRemoteFile(options.Dest.Filepath)
	ex, err := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

	file_level.CheckErr(err)

	resp := ex.Search()

	rf.WriteSyncedFile(&resp, options.Dest.Filepath, true)

	return nil
}

func ExecuteTCPExchange(options *Options) error {
	return transport.SendFile(options.Source.Filepath, options.Source.Address+":42069")
}

func ExecuteStartServer(options *Options) error {
	serv, err := transport.StartServer(42069)
	if err != nil {
		panic(err)
	}
	return serv.Run()
}
