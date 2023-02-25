package transport

import (
	"fmt"
)

// first request client ---> server
type InitialFileRequest struct {
	Filename string
	Md5sum   [16]byte
}

type PacketType int

const (
	INFO_PACKET PacketType = iota
	DATA_PACKET
	ERROR_PACKET
)

type StatusResponse int

const (
	STATUS_FILE_SYNCED StatusResponse = iota
	STATUS_FILE_EXISTS
	STATUS_REQUEST_CHUNKS
	STATUS_SENDING_CHUNKS
	STATUS_SERVER_ERROR
)

type StatusMessages struct {
	Status  StatusResponse
	Message string
}

func (sm StatusMessages) String() (str string) {
	return fmt.Sprintf(
		"Status: <%d>, Message <%s>\n",
		sm.Status, sm.Message,
	)
}

func (ifr InitialFileRequest) String() (str string) {
	return fmt.Sprintf(
		"Filename: <%v>, MD5 <%v>\n",
		ifr.Filename, ifr.Md5sum,
	)
}

func (status StatusResponse) String() (str string) {
	switch status {
	case STATUS_FILE_SYNCED:
		return "STATUS_FILE_SYNCED"
	case STATUS_FILE_EXISTS:
		return "STATUS_FILE_EXISTS"
	case STATUS_REQUEST_CHUNKS:
		return "STATUS_REQUEST_CHUNKS"
	case STATUS_SENDING_CHUNKS:
		return "STATUS_SENDING_CHUNKS"
	case STATUS_SERVER_ERROR:
		return "STATUS_SERVER_ERROR"
	default:
		return fmt.Sprintf("%d", status)
	}
}
