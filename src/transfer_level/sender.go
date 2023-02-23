package transport

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"

	"github.com/andreistan26/sync/src/file_level"
)

func SendFile(filename string, addr string) error {
	sourceFile := file_level.CreateSourceFile(filename)

	netConn, err := net.Dial("tcp4", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		panic(err)
	}

	conn := SyncConn{
		Encoder: gob.NewEncoder(netConn),
		Decoder: gob.NewDecoder(netConn),
	}

	md5sum, err := file_level.GetFileMD5(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		panic(err)
	}

	conn.Encode(InitialFileRequest{
		Filename: filename,
		Md5sum:   md5sum,
	})

	fmt.Println(filename, " ", md5sum)

	var statusMsg StatusMessages
	conn.Decode(&statusMsg)

	//TODO
	if statusMsg.Status != STATUS_SENDING_CHUNKS {
		fmt.Fprintf(os.Stderr, "TODO implement this")
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
	return nil
}
