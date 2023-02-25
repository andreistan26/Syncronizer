package transport

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

type SyncConn struct {
	RW      *bufio.ReadWriter
	Encoder *gob.Encoder
	Decoder *gob.Decoder
}

func InitSyncConn(conn net.Conn) (syncConn *SyncConn) {
	syncConn = &SyncConn{}
	syncConn.Encoder = gob.NewEncoder(conn)
	syncConn.Decoder = gob.NewDecoder(conn)
	return syncConn
}

func (conn *SyncConn) Decode(e any) error {
	if _, ok := e.(StatusMessages); ok {
		log.Println(e.(StatusMessages))
	}
	err := conn.Decoder.Decode(e)
	if err != nil {
		fmt.Printf("Error from Decode : %v\n", err)
	}
	return err
}

func (conn *SyncConn) Encode(e any) error {
	if _, ok := e.(StatusMessages); ok {
		log.Println(e.(StatusMessages))
	}
	err := conn.Encoder.Encode(e)
	if err != nil {
		fmt.Printf("Error from Encode : %v\n", err)
	}
	return err
}
