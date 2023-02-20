package cmd

import "github.com/andreistan26/sync/src/file_level"

func Execute(options Options) error {
	return ExecuteHostExchange(options)
}

func ExecuteHostExchange(options Options) error {
	sf := file_level.CreateSourceFile(options.Source)
	rf := file_level.CreateRemoteFile(options.Dest)
	ex, err := file_level.CreateRsyncExchange(&sf, rf.ChunkList)

	file_level.CheckErr(err)

	resp := ex.Search()

	rf.WriteSyncedFile(&resp, options.Dest, true)

	return nil
}
