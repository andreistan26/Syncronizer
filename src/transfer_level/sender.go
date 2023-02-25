package transport

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/andreistan26/sync/src/file_level"
	"github.com/andreistan26/sync/src/options"
)

func SendFile(opts *options.Options) error {
	sourceFile := file_level.CreateSourceFile(opts.Source.Filepath)

	netConn, err := net.Dial("tcp4", opts.Dest.Address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		panic(err)
	}

	conn := InitSyncConn(netConn)

	md5sum, err := file_level.GetFileMD5(opts.Source.Filepath)
	if err != nil {
		log.Printf("Error occured when calculating md5 for file %v\n", err)
		panic(err)
	}

	conn.Encode(InitialFileRequest{
		Filename: opts.Dest.Filepath,
		Md5sum:   md5sum,
	})

	var statusMsg StatusMessages
	conn.Decode(&statusMsg)

	if statusMsg.Status != STATUS_SENDING_CHUNKS {
		log.Println(statusMsg)
	}

	var remoteChunkList []file_level.Chunk
	conn.Decode(&remoteChunkList)

	ex, err := file_level.CreateRsyncExchange(&sourceFile, remoteChunkList)
	if err != nil {
		panic(err)
	}

	resp := ex.Search()
	conn.Encode(resp)

	conn.Decode(&statusMsg)
	if statusMsg.Status == STATUS_FILE_SYNCED {
		fmt.Println("File sync succesful!")
	}

	netConn.Close()
	return nil
}
