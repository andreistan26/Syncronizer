package transport

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"reflect"

	"github.com/andreistan26/sync/src/file_level"
)

type SyncServerTCP struct {
	Addr    net.Addr
	Listner net.Listener
}

func StartServer(port int) (serv *SyncServerTCP, err error) {
	serv = &SyncServerTCP{}
	serv.Listner, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))

	if err != nil {
		log.Printf("Error occured when starting server : %v", err)
	}
	return serv, err
}

func (serv *SyncServerTCP) Run() error {
	for {
		conn, err := serv.Listner.Accept()
		if err != nil {
			log.Printf("Got error %v when Accepting\n", err)
		}

		fmt.Fprintf(os.Stderr, "TCP connection established with %v\n", conn.RemoteAddr().String())
		syncConn := InitSyncConn(conn)
		go func() {
			defer conn.Close()
			syncConn.HandleConnection()
		}()
	}
}

// TODO investigate behavior if file is open by a different process
func (conn *SyncConn) HandleConnection() error {
	// wait for fliepath and checksum
	initialFileRequest := &InitialFileRequest{}
	err := conn.Decode(initialFileRequest)
	if err != nil {
		log.Println(initialFileRequest.Filename, " ", initialFileRequest.Md5sum)
		fmt.Printf("Got an error when trying to decode initial file request\n")
		return err
	}

	// probe hash in order to check if the file is unmodified
	md5, err := file_level.GetFileMD5(initialFileRequest.Filename)

	// file exists, md5 crashed
	if _, ok := err.(*os.PathError); err != nil && !ok {
		conn.Encode(StatusMessages{
			Status:  STATUS_SERVER_ERROR,
			Message: "Calculating md5sum error",
		})
		fmt.Fprintf(os.Stderr, "Got an error from md5 function that is not path related, %v", err)
	} else if ok {
		// TODO add config if path is not in system to make or abort
		// file does not exist, just copy it
		dirPath := path.Join(initialFileRequest.Filename, "..")
		os.MkdirAll(dirPath, os.ModePerm)
	}

	// files are the same
	if reflect.DeepEqual(md5, initialFileRequest.Md5sum) {
		conn.Encode(StatusMessages{
			Status:  STATUS_FILE_EXISTS,
			Message: "File already exists",
		})
		return nil
	}

	// file exists but is modified
	conn.Encode(StatusMessages{
		Status:  STATUS_SENDING_CHUNKS,
		Message: "Sending Chunks",
	})

	// send chunks of data
	remoteFile := file_level.CreateRemoteFile(initialFileRequest.Filename)
	conn.Encode(remoteFile.ChunkList)

	// waiting for reponse package
	var response file_level.Response
	conn.Decode(&response)
	log.Printf("%v", response)
	remoteFile.WriteSyncedFile(&response, initialFileRequest.Filename, true)

	resultMD5, err := file_level.GetFileMD5(initialFileRequest.Filename)
	if err != nil {
		log.Printf("Error occured when calculating md5 on final file, %v\n", err)
	}
	if reflect.DeepEqual(resultMD5, initialFileRequest.Md5sum) {
		conn.Encode(StatusMessages{
			Status:  STATUS_FILE_SYNCED,
			Message: "file synced (msg from server)",
		})
	}
	return nil

}
